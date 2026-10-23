package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

// Minecraft 局域网发现：
//   - Java 版「对局域网开放」后每 1.5 秒向 224.0.2.60:4445 组播 "[MOTD]世界名[/MOTD][AD]端口[/AD]"，
//     端口是随机的，只能靠监听组播得知；
//   - 基岩版服务器响应 RakNet Unconnected Ping（默认端口 19132），Pong 中带有服务器名与 IPv4 端口。

const (
	minecraftJavaLANGroup   = "224.0.2.60:4445"
	minecraftJavaListenTime = 1600 * time.Millisecond // 略长于 1.5 秒的广播间隔
	minecraftBedrockPort    = 19132
	minecraftBedrockTimeout = 800 * time.Millisecond

	ServiceMinecraftJava    = "minecraft-java"
	ServiceMinecraftBedrock = "minecraft-bedrock"
)

var (
	minecraftJavaAdPattern = regexp.MustCompile(`\[MOTD\](.*?)\[/MOTD\]\[AD\](\d+)\[/AD\]`)
	// RakNet 离线消息魔数
	raknetMagic = []byte{0x00, 0xff, 0xff, 0x00, 0xfe, 0xfe, 0xfe, 0xfe, 0xfd, 0xfd, 0xfd, 0xfd, 0x12, 0x34, 0x56, 0x78}
)

// minecraftServer 是通过局域网发现的 Minecraft 服务器。
type minecraftServer struct {
	service  string // ServiceMinecraftJava / ServiceMinecraftBedrock
	protocol string
	ip       net.IP // 发送方地址
	port     int
	motd     string
}

// discoverMinecraftServers 并行进行 Java 组播监听与基岩版探测，最长约 minecraftJavaListenTime。
func discoverMinecraftServers(ctx context.Context) []minecraftServer {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		servers []minecraftServer
	)
	collect := func(found []minecraftServer) {
		mu.Lock()
		servers = append(servers, found...)
		mu.Unlock()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		collect(listenMinecraftJavaLAN(ctx, minecraftJavaListenTime))
	}()
	go func() {
		defer wg.Done()
		collect(pingMinecraftBedrock(ctx, minecraftBedrockTimeout))
	}()
	wg.Wait()
	return servers
}

func listenMinecraftJavaLAN(ctx context.Context, duration time.Duration) []minecraftServer {
	group, err := net.ResolveUDPAddr("udp4", minecraftJavaLANGroup)
	if err != nil {
		return nil
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return nil
	}
	defer func() {
		_ = conn.Close()
	}()
	setupMulticastListener(conn, group.IP)

	deadline := time.Now().Add(duration)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetReadDeadline(deadline)

	seen := map[string]bool{}
	var servers []minecraftServer
	buf := make([]byte, 1024)
	for {
		n, sender, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		match := minecraftJavaAdPattern.FindSubmatch(buf[:n])
		if match == nil {
			continue
		}
		port, err := strconv.Atoi(string(match[2]))
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		key := sender.IP.String() + ":" + strconv.Itoa(port)
		if seen[key] {
			continue
		}
		seen[key] = true
		servers = append(servers, minecraftServer{
			service:  ServiceMinecraftJava,
			protocol: "tcp",
			ip:       sender.IP,
			port:     port,
			motd:     stripMinecraftFormatting(string(match[1])),
		})
	}
	return servers
}

// pingMinecraftBedrock 向本机与局域网广播发送 RakNet Unconnected Ping，收集基岩版服务器的 Pong。
func pingMinecraftBedrock(ctx context.Context, timeout time.Duration) []minecraftServer {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil
	}
	defer func() {
		_ = conn.Close()
	}()

	ping := make([]byte, 0, 33)
	ping = append(ping, 0x01)
	ping = binary.BigEndian.AppendUint64(ping, uint64(time.Now().UnixMilli()))
	ping = append(ping, raknetMagic...)
	guid := make([]byte, 8)
	_, _ = rand.Read(guid)
	ping = append(ping, guid...)

	targets := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4bcast}
	for _, target := range targets {
		_, _ = conn.WriteToUDP(ping, &net.UDPAddr{IP: target, Port: minecraftBedrockPort})
	}

	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetReadDeadline(deadline)

	seen := map[string]bool{}
	var servers []minecraftServer
	buf := make([]byte, 2048)
	for {
		n, sender, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		motd, port, ok := parseBedrockPong(buf[:n])
		if !ok {
			continue
		}
		if port == 0 {
			port = sender.Port
		}
		key := sender.IP.String() + ":" + strconv.Itoa(port)
		if seen[key] {
			continue
		}
		seen[key] = true
		servers = append(servers, minecraftServer{
			service:  ServiceMinecraftBedrock,
			protocol: "udp",
			ip:       sender.IP,
			port:     port,
			motd:     motd,
		})
	}
	return servers
}

// parseBedrockPong 解析 Unconnected Pong：0x1c + 时间(8) + 服务器 GUID(8) + 魔数(16) + 长度(2) + 字符串，
// 字符串形如 "MCPE;服务器名;协议;版本;在线;上限;服务器ID;世界名;模式;1;IPv4端口;IPv6端口;"。
func parseBedrockPong(packet []byte) (string, int, bool) {
	const header = 1 + 8 + 8 + 16 + 2
	if len(packet) < header || packet[0] != 0x1c || !bytes.Equal(packet[17:33], raknetMagic) {
		return "", 0, false
	}
	length := int(binary.BigEndian.Uint16(packet[33:35]))
	if len(packet) < header+length {
		return "", 0, false
	}
	fields := strings.Split(string(packet[header:header+length]), ";")
	if len(fields) < 2 || (fields[0] != "MCPE" && fields[0] != "MCEE") {
		return "", 0, false
	}
	motd := stripMinecraftFormatting(fields[1])
	if len(fields) > 7 && strings.TrimSpace(fields[7]) != "" {
		motd = strings.TrimSpace(motd + " · " + stripMinecraftFormatting(fields[7]))
	}
	port := 0
	if len(fields) > 10 {
		if value, err := strconv.Atoi(fields[10]); err == nil && value > 0 && value <= 65535 {
			port = value
		}
	}
	return motd, port, true
}

// stripMinecraftFormatting 去掉 § 格式代码（颜色、粗体等）。
func stripMinecraftFormatting(text string) string {
	var b strings.Builder
	skip := false
	for _, r := range text {
		if skip {
			skip = false
			continue
		}
		if r == '§' {
			skip = true
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// isLocalIP 判断地址是否属于本机（回环或本机网卡地址）。
func isLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.Equal(ip) {
			return true
		}
	}
	return false
}

// setupMulticastListener 让组播监听能收到本机游戏发出的广播：
//   - 重新开启 IP_MULTICAST_LOOP：ListenMulticastUDP 会将其关闭。该选项在 Linux / macOS 上只影响本套接字
//     发出的组播，但在 Windows（Winsock）上作用于接收方向——关闭后收不到本机任何进程发出的组播，
//     游戏与本应用在同一台 Windows 电脑上时就扫描不到；
//   - 在所有启用的多播网卡上加入组播组：ListenMulticastUDP(nil) 只在系统默认网卡上加入，而该网卡常是
//     虚拟网卡（Hyper-V、WSL、VPN / TUN 等），游戏却从实体网卡发出广播。已加入的网卡会返回错误，忽略即可。
func setupMulticastListener(conn *net.UDPConn, group net.IP) {
	packetConn := ipv4.NewPacketConn(conn)
	_ = packetConn.SetMulticastLoopback(true)

	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for i := range interfaces {
		iface := interfaces[i]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		_ = packetConn.JoinGroup(&iface, &net.UDPAddr{IP: group})
	}
}
