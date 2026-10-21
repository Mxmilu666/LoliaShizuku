// Command icons generates the Android launcher icons (mipmap-*) from build/appicon.png.
//
// Usage: go run build/android/scripts/icons/main.go [input] [resDir]
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// 各密度下的启动图标尺寸（dp 48 × 缩放系数）
var densities = map[string]int{
	"mipmap-mdpi":    48,
	"mipmap-hdpi":    72,
	"mipmap-xhdpi":   96,
	"mipmap-xxhdpi":  144,
	"mipmap-xxxhdpi": 192,
}

func main() {
	input := "build/appicon.png"
	resDir := "build/android/app/src/main/res"
	if len(os.Args) > 1 {
		input = os.Args[1]
	}
	if len(os.Args) > 2 {
		resDir = os.Args[2]
	}

	src, err := loadPNG(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read icon:", err)
		os.Exit(1)
	}

	for dir, size := range densities {
		resized := resize(src, size)
		for _, name := range []string{"ic_launcher.png", "ic_launcher_round.png"} {
			out := filepath.Join(resDir, dir, name)
			if err := writePNG(out, resized); err != nil {
				fmt.Fprintln(os.Stderr, "write icon:", err)
				os.Exit(1)
			}
		}
	}
}

func loadPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

// resize 以区域平均（box filter）缩放到 size×size，适合大图缩小
func resize(src image.Image, size int) image.Image {
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		y0 := bounds.Min.Y + y*srcH/size
		y1 := max(y0+1, bounds.Min.Y+(y+1)*srcH/size)
		for x := 0; x < size; x++ {
			x0 := bounds.Min.X + x*srcW/size
			x1 := max(x0+1, bounds.Min.X+(x+1)*srcW/size)
			var r, g, b, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					c := color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA)
					r += uint64(c.R)
					g += uint64(c.G)
					b += uint64(c.B)
					a += uint64(c.A)
					n++
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n)})
		}
	}
	return dst
}
