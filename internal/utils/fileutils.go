package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileUtils struct{}

func NewFileUtils() *FileUtils {
	return &FileUtils{}
}

func (fu *FileUtils) ExtractFileFromVSIX(vsixPath, filePath string) ([]byte, error) {
	reader, err := zip.OpenReader(vsixPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .vsix file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == filePath {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			return content, nil
		}
	}

	return nil, fmt.Errorf("file %s not found in .vsix archive", filePath)
}

func (fu *FileUtils) DetectContentType(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return OctetStreamContentType
	}
	defer file.Close()

	buffer := make([]byte, 512)
	bytesRead, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return OctetStreamContentType
	}

	contentType := detectContentType(buffer[:bytesRead])

	if strings.Contains(string(buffer[:bytesRead]), "<?xml") ||
		strings.Contains(string(buffer[:bytesRead]), "<svg") {
		return "image/svg+xml; charset=utf-8"
	}

	return contentType
}

func (fu *FileUtils) GetMimeTypeByExtension(filename string) string {
	fileExt := strings.ToLower(filepath.Ext(filename))

	switch fileExt {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".ico":
		return "image/x-icon"
	case ".webp":
		return "image/webp"
	default:
		return OctetStreamContentType
	}
}

func (fu *FileUtils) EnsureDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

func (fu *FileUtils) IsVSIXFile(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".vsix")
}

func (fu *FileUtils) GetFileName(filePath string) string {
	return filepath.Base(filePath)
}

func (fu *FileUtils) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func detectContentType(data []byte) string {
	if len(data) == 0 {
		return OctetStreamContentType
	}

	if len(data) >= 2 {
		switch {
		case data[0] == 0xFF && data[1] == 0xD8:
			return "image/jpeg"
		case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
			return "image/png"
		case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
			return "image/gif"
		}
	}

	if isText(data) {
		return "text/plain; charset=utf-8"
	}

	return OctetStreamContentType
}

func isText(data []byte) bool {
	for _, b := range data {
		if b < 0x20 && b != 0x09 && b != 0x0A && b != 0x0D {
			return false
		}
	}
	return true
}
