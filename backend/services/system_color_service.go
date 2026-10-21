package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// wallpaperSampleSize 壁纸缩略图的最长边。取色只需要颜色分布，128px 足够且计算量小。
const wallpaperSampleSize = 128

// WallpaperPixels 是缩小后的壁纸像素，供前端按 MD3（莫奈）算法提取种子色。
type WallpaperPixels struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	RGBA   string `json:"rgba"` // base64 编码的 RGBA 像素
}

// SystemColorService 提供动态取色所需的系统颜色信息。
type SystemColorService struct{}

func NewSystemColorService() *SystemColorService {
	return &SystemColorService{}
}

// GetWallpaperPixels 读取当前桌面壁纸并缩小为 RGBA 像素。
func (s *SystemColorService) GetWallpaperPixels() (*WallpaperPixels, error) {
	path, err := wallpaperPath()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("无法获取桌面壁纸")
	}

	img, err := decodeWallpaper(path)
	if err != nil {
		return nil, fmt.Errorf("读取桌面壁纸失败: %w", err)
	}

	width, height, rgba := sampleRGBA(img, wallpaperSampleSize)
	if width == 0 || height == 0 {
		return nil, errors.New("桌面壁纸为空")
	}
	return &WallpaperPixels{
		Width:  width,
		Height: height,
		RGBA:   base64.StdEncoding.EncodeToString(rgba),
	}, nil
}

// GetSystemAccentColor 返回系统强调色（CSS 颜色字符串），无法获取时返回空字符串。
func (s *SystemColorService) GetSystemAccentColor() string {
	if color := platformAccentColor(); color != "" {
		return color
	}
	app := application.Get()
	if app == nil || app.Env == nil {
		return ""
	}
	return app.Env.GetAccentColor()
}

func decodeImageFile(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	img, _, err := image.Decode(file)
	return img, err
}

// sampleRGBA 以最近邻采样把图片缩小到最长边不超过 maxSize。
func sampleRGBA(img image.Image, maxSize int) (int, int, []byte) {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return 0, 0, nil
	}

	dstW, dstH := srcW, srcH
	if srcW > maxSize || srcH > maxSize {
		if srcW >= srcH {
			dstW = maxSize
			dstH = max(1, srcH*maxSize/srcW)
		} else {
			dstH = maxSize
			dstW = max(1, srcW*maxSize/srcH)
		}
	}

	out := make([]byte, 0, dstW*dstH*4)
	for y := 0; y < dstH; y++ {
		sy := bounds.Min.Y + y*srcH/dstH
		for x := 0; x < dstW; x++ {
			sx := bounds.Min.X + x*srcW/dstW
			r, g, b, a := img.At(sx, sy).RGBA()
			out = append(out, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	return dstW, dstH, out
}
