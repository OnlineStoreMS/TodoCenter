package admin

import (
	"mime/multipart"
	"path/filepath"
	"strings"
)

const (
	maxImageUploadSize = 10 << 20  // 10MB
	maxVideoUploadSize = 100 << 20 // 100MB
)

func classifyUploadFile(file *multipart.FileHeader) (kind string, maxSize int64, ok bool) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	ct := strings.ToLower(file.Header.Get("Content-Type"))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".heic":
		return "image", maxImageUploadSize, true
	case ".mp4", ".mov", ".webm", ".m4v", ".avi", ".mkv":
		return "video", maxVideoUploadSize, true
	}
	if strings.HasPrefix(ct, "image/") {
		return "image", maxImageUploadSize, true
	}
	if strings.HasPrefix(ct, "video/") {
		return "video", maxVideoUploadSize, true
	}
	return "", 0, false
}
