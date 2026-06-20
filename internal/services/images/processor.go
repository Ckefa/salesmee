package images

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deepteams/webp"
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
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("images: create %s: %w", outputPath, err)
		}
		defer f.Close()
		if err := webp.Encode(f, resized, &webp.EncoderOptions{
			Quality: cfg.Quality,
			Method:  4,
		}); err != nil {
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

func RemoveIfExists(path string) {
	if path != "" {
		os.Remove(path)
	}
}

func IsImageType(mediaType string) bool {
	switch mediaType {
	case "image", "jpg", "jpeg", "png", "gif", "webp":
		return true
	}
	return false
}


