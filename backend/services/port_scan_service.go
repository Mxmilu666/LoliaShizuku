package services

import (
	"context"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	portScanTimeout  = 8 * time.Second
	portProbeTimeout = 300 * time.Millisecond
)

// 系统列表拿不到的监听通过本机 TCP 连接探测补充：macOS 上 root 进程的端口（lsof 无权查看），
// 以及 Android 10+（应用无权读取 /proc/net，只能探测；8022 为 Termux 的 SSH）
var commonProbePorts = []int{
	21, 22, 80, 443, 1433, 3000, 3306, 3389, 5000, 5173, 5432, 5900, 6379, 7777,
	8000, 8022, 8080, 8081, 8088, 8443, 8888, 9000, 9090, 25565, 25575, 27015, 27017,
}

// 系统常驻、几乎不会用来做隧道的 UDP 端口（mDNS、SSDP、NetBIOS、DHCP、NTP、LLMNR）
var noisyUDPPorts = map[int]bool{67: true, 68: true, 123: true, 137: true, 138: true, 1900: true, 5353: true, 5355: true}

// UDP 客户端套接字多绑定在临时端口上，不视为服务
const udpEphemeralPortStart = 32768

// ListeningPort 本机正在监听的端口。
type ListeningPort struct {
	Protocol string `json:"protocol"` // tcp / udp
	Port     int    `json:"port"`
	Address  string `json:"address"`  // 绑定地址
	LocalIP  string `json:"local_ip"` // 建议填写到隧道的本地 IP
	PID      int32  `json:"pid"`
	Process  string `json:"process"`
	Probed   bool   `json:"probed"`  // 由连接探测发现（无进程信息）
	Service  string `json:"service"` // 通过协议识别出的服务，如 minecraft-java / minecraft-bedrock
	Title    string `json:"title"`   // 服务自报的名称，如 Minecraft 世界名 / MOTD
	Remote   bool   `json:"remote"`  // 位于局域网内其他设备
}

// PortScanService 扫描本机监听端口，供新建隧道时选择本地服务。
type PortScanService struct{}

func NewPortScanService() *PortScanService {
	return &PortScanService{}
}

// ListListeningPorts 返回本机监听中的 TCP / UDP 端口（同协议同端口合并），按端口排序。
func (s *PortScanService) ListListeningPorts() ([]ListeningPort, error) {
	ctx, cancel := context.WithTimeout(context.Background(), portScanTimeout)
	defer cancel()

	// Minecraft 局域网发现需监听约 1.6 秒，与端口列表并行进行
	minecraftCh := make(chan []minecraftServer, 1)
	go func() {
		minecraftCh <- discoverMinecraftServers(ctx)
	}()

	// 读取失败（如 Android 无权读取 /proc/net）时仅依靠下方的连接探测
	conns, _ := psnet.ConnectionsWithContext(ctx, "inet")
	selfPID := int32(os.Getpid())

	type entry struct {
		port      ListeningPort
		addresses []string
	}
	merged := map[string]*entry{}
	for _, conn := range conns {
		protocol, ok := listeningProtocol(conn)
		if !ok || conn.Pid == selfPID {
			continue
		}
		port := int(conn.Laddr.Port)
		if port <= 0 || (protocol == "udp" && (port >= udpEphemeralPortStart || noisyUDPPorts[port])) {
			continue
		}
		key := protocol + "/" + strconv.Itoa(port)
		item, exists := merged[key]
		if !exists {
			item = &entry{port: ListeningPort{Protocol: protocol, Port: port, PID: conn.Pid}}
			merged[key] = item
		}
		if item.port.PID == 0 {
			item.port.PID = conn.Pid
		}
		item.addresses = append(item.addresses, conn.Laddr.IP)
	}

	pids := make([]int32, 0, len(merged))
	for _, item := range merged {
		pids = append(pids, item.port.PID)
	}
	names := processNames(ctx, pids)
	result := make([]ListeningPort, 0, len(merged))
	for _, item := range merged {
		item.port.Address, item.port.LocalIP = pickLocalAddress(item.addresses)
		item.port.Process = names[item.port.PID]
		result = append(result, item.port)
	}

	// 补充探测：系统列表中没有的常见 TCP 端口
	known := map[int]bool{}
	for _, item := range result {
		if item.Protocol == "tcp" {
			known[item.Port] = true
		}
	}
	result = append(result, probeLocalTCPPorts(known)...)
	result = mergeMinecraftServers(result, <-minecraftCh)

	sort.Slice(result, func(i, j int) bool {
		if result[i].Port != result[j].Port {
			return result[i].Port < result[j].Port
		}
		return result[i].Protocol < result[j].Protocol
	})
	return result, nil
}

// listeningProtocol 判断连接是否为监听中的服务端套接字：TCP 为 LISTEN；UDP 无状态，取未连接的套接字。
func listeningProtocol(conn psnet.ConnectionStat) (string, bool) {
	switch conn.Type {
	case 1: // SOCK_STREAM
		return "tcp", strings.EqualFold(conn.Status, "LISTEN")
	case 2: // SOCK_DGRAM
		return "udp", conn.Raddr.Port == 0
	}
	return "", false
}

// pickLocalAddress 返回展示用的绑定地址，以及 frpc 连接本地服务应使用的 IP。
func pickLocalAddress(addresses []string) (string, string) {
	var v4Wildcard, v6Wildcard, v4Loopback, v6Loopback bool
	var specific string
	for _, address := range addresses {
		ip := net.ParseIP(strings.Trim(address, "[]"))
		switch {
		case address == "*" || address == "0.0.0.0":
			v4Wildcard = true
		case address == "::" || (ip != nil && ip.IsUnspecified()):
			v6Wildcard = true
		case ip != nil && ip.IsLoopback() && ip.To4() != nil:
			v4Loopback = true
		case ip != nil && ip.IsLoopback():
			v6Loopback = true
		case ip != nil && specific == "":
			specific = ip.String()
		}
	}
	switch {
	case v4Wildcard:
		return "0.0.0.0", "127.0.0.1"
	case v6Wildcard && runtime.GOOS == "windows":
		// Windows 的 [::] 监听通常仅限 IPv6
		return "::", "::1"
	case v6Wildcard:
		// Linux / macOS 默认双栈，[::] 同时接受 IPv4
		return "::", "127.0.0.1"
	case v4Loopback:
		return "127.0.0.1", "127.0.0.1"
	case v6Loopback:
		return "::1", "::1"
	case specific != "":
		return specific, specific
	}
	return "", "127.0.0.1"
}

// processNames 查询进程名；无权限读取的进程（如其他用户 / 其他应用）返回空字符串。
func processNames(ctx context.Context, pids []int32) map[int32]string {
	names := map[int32]string{}
	for _, pid := range pids {
		if _, ok := names[pid]; ok || pid <= 0 {
			continue
		}
		name := ""
		if proc, err := process.NewProcessWithContext(ctx, pid); err == nil {
			name, _ = proc.NameWithContext(ctx)
		}
		names[pid] = strings.TrimSuffix(name, ".exe")
	}
	return names
}

func probeLocalTCPPorts(known map[int]bool) []ListeningPort {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result []ListeningPort
	)
	for _, port := range commonProbePorts {
		if known[port] {
			continue
		}
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), portProbeTimeout)
			if err != nil {
				return
			}
			_ = conn.Close()
			mu.Lock()
			result = append(result, ListeningPort{
				Protocol: "tcp",
				Port:     port,
				Address:  "127.0.0.1",
				LocalIP:  "127.0.0.1",
				Probed:   true,
			})
			mu.Unlock()
		}(port)
	}
	wg.Wait()
	return result
}

// mergeMinecraftServers 将发现的 Minecraft 服务器合并到端口列表：本机已有的端口补充服务信息，否则新增一项。
func mergeMinecraftServers(ports []ListeningPort, servers []minecraftServer) []ListeningPort {
	for _, server := range servers {
		local := isLocalIP(server.ip)
		merged := false
		if local {
			for i := range ports {
				if ports[i].Protocol == server.protocol && ports[i].Port == server.port && !ports[i].Remote {
					ports[i].Service = server.service
					ports[i].Title = server.motd
					merged = true
					break
				}
			}
		}
		if merged {
			continue
		}
		item := ListeningPort{
			Protocol: server.protocol,
			Port:     server.port,
			Address:  server.ip.String(),
			LocalIP:  server.ip.String(),
			Service:  server.service,
			Title:    server.motd,
			Remote:   !local,
		}
		if local {
			item.LocalIP = "127.0.0.1"
		}
		ports = append(ports, item)
	}
	return ports
}
