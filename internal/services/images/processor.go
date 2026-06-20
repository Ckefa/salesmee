package images

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

type Config struct {
	MaxWidth  int
	MaxHeight int
	Quality   float32
}

var DefaultConfig = Config{
	MaxWidth:  1200,
	MaxHeight: 1200,
	Quality:   80,
}

var LogoConfig = Config{
	MaxWidth:  400,
	MaxHeight: 400,
	Quality:   80,
}

func Process(inputPath, outputPath string, cfg Config) error {
	if cfg.Quality <= 0 {
		cfg.Quality = DefaultConfig.Quality
	}

	src, err := imaging.Open(inputPath)
	if err != nil {
		return fmt.Errorf("images: open %s: %w", inputPath, err)
	}

	resized := imaging.Fit(src, cfg.MaxWidth, cfg.MaxHeight, imaging.Lanczos)

	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("images: mkdir %s: %w", outDir, err)
	}

	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".webp":
		rgba := toRGBA(resized)
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("images: create %s: %w", outputPath, err)
		}
		defer f.Close()

		if err := webp.Encode(f, rgba, &webp.Options{Quality: cfg.Quality}); err != nil {
			return fmt.Errorf("images: encode webp %s: %w", outputPath, err)
		}
	case ".jpg", ".jpeg":
		if err := imaging.Save(resized, outputPath, imaging.JPEGQuality(int(cfg.Quality))); err != nil {
			return fmt.Errorf("images: save jpeg %s: %w", outputPath, err)
		}
	case ".png":
		if err := imaging.Save(resized, outputPath); err != nil {
			return fmt.Errorf("images: save png %s: %w", outputPath, err)
		}
	default:
		return fmt.Errorf("images: unsupported output format: %s", ext)
	}

	os.Remove(inputPath)

	return nil
}

func ProcessAsWebP(inputPath, outputWebPPath string, cfg Config) error {
	return Process(inputPath, outputWebPPath, cfg)
}

func RemoveIfExists(path string) {
	if path != "" {
		os.Remove(path)
	}
}

func toRGBA(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			rgba.SetRGBA(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}
	return rgba
}

func IsImageType(mediaType string) bool {
	switch mediaType {
	case "image", "jpg", "jpeg", "png", "gif", "webp":
		return true
	}
	return false
}


