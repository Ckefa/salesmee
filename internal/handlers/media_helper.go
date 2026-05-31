package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func SaveMediaFile(c *gin.Context, formField string) (string, string, error) {
	file, header, err := c.Request.FormFile(formField)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	var mediaType string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		mediaType = "image"
	case ".pdf", ".doc", ".docx", ".txt", ".xls", ".xlsx", ".csv":
		mediaType = "document"
	case ".mp3", ".wav", ".ogg", ".m4a", ".aac", ".webm":
		mediaType = "audio"
	default:
		return "", "", fmt.Errorf("unsupported file type: %s", ext)
	}

	if header.Size > 10*1024*1024 {
		return "", "", fmt.Errorf("file too large: %d bytes", header.Size)
	}

	uploadDir := filepath.Join("web", "static", "uploads", "media")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", "", err
	}

	filename := fmt.Sprintf("%s_%d%s", mediaType, time.Now().UnixNano(), ext)
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return "", "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", "", err
	}

	mediaURL := filepath.Join("uploads", "media", filename)
	return mediaURL, mediaType, nil
}
