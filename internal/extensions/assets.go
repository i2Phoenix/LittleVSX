package extensions

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AssetProcessor struct {
	assetsDir string
	baseURL   string
}

func NewAssetProcessor(assetsDir, baseURL string) *AssetProcessor {
	return &AssetProcessor{
		assetsDir: assetsDir,
		baseURL:   baseURL,
	}
}

func (ap *AssetProcessor) ProcessReadme(readmeContent, extensionID string) (string, error) {
	if readmeContent == "" {
		return "", nil
	}

	extensionAssetsDir := filepath.Join(ap.assetsDir, extensionID)
	if err := os.MkdirAll(extensionAssetsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create asset directory: %w", err)
	}

	processedContent := ap.processImages(readmeContent, extensionAssetsDir, extensionID)
	processedContent = ap.processOtherAssets(processedContent, extensionAssetsDir, extensionID)

	return processedContent, nil
}

func (ap *AssetProcessor) processImages(content, assetsDir, extensionID string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`),
		regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`),
		regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\s*"([^"]*)"\)`),
	}

	for _, pattern := range patterns {
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			return ap.processImageMatch(match, pattern, assetsDir, extensionID)
		})
	}

	return content
}

func (ap *AssetProcessor) processImageMatch(match string, pattern *regexp.Regexp, assetsDir, extensionID string) string {
	matches := pattern.FindStringSubmatch(match)
	if len(matches) < 2 {
		return match
	}

	var imageURL string
	if len(matches) >= 3 {
		imageURL = matches[2]
	} else {
		imageURL = matches[1]
	}

	if strings.HasPrefix(imageURL, "data:") || strings.HasPrefix(imageURL, "#") {
		return match
	}

	localPath, err := ap.downloadAsset(imageURL, assetsDir)
	if err != nil {
		fmt.Printf("Failed to download image %s: %v\n", imageURL, err)
		return match
	}

	localURL := fmt.Sprintf("%s/_assets/%s/%s", ap.baseURL, extensionID, filepath.Base(localPath))

	if strings.Contains(match, "![") {
		if len(matches) >= 3 {
			return fmt.Sprintf("![%s](%s)", matches[1], localURL)
		}
		return fmt.Sprintf("![](%s)", localURL)
	} else {
		return strings.Replace(match, imageURL, localURL, 1)
	}
}

func (ap *AssetProcessor) processOtherAssets(content, assetsDir, extensionID string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<link[^>]+href=["']([^"']+)["'][^>]*>`),
		regexp.MustCompile(`<script[^>]+src=["']([^"']+)["'][^>]*>`),
		regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`),
	}

	for _, pattern := range patterns {
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			return ap.processAssetMatch(match, pattern, assetsDir, extensionID)
		})
	}

	return content
}

func (ap *AssetProcessor) processAssetMatch(match string, pattern *regexp.Regexp, assetsDir, extensionID string) string {
	matches := pattern.FindStringSubmatch(match)
	if len(matches) < 2 {
		return match
	}

	var assetURL string
	if len(matches) >= 3 {
		assetURL = matches[2]
	} else {
		assetURL = matches[1]
	}

	if strings.HasPrefix(assetURL, "data:") ||
		strings.HasPrefix(assetURL, "#") ||
		strings.HasPrefix(assetURL, "http") {
		return match
	}

	localPath, err := ap.downloadAsset(assetURL, assetsDir)
	if err != nil {
		fmt.Printf("Failed to download asset %s: %v\n", assetURL, err)
		return match
	}

	localURL := fmt.Sprintf("%s/_assets/%s/%s", ap.baseURL, extensionID, filepath.Base(localPath))

	if strings.Contains(match, "<link") || strings.Contains(match, "<script") {
		return strings.Replace(match, assetURL, localURL, 1)
	} else {
		return fmt.Sprintf("[%s](%s)", matches[1], localURL)
	}
}

func (ap *AssetProcessor) downloadAsset(assetURL, assetsDir string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(assetURL)
	if err != nil {
		return "", fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	fileName := ap.generateFileName(assetURL, resp.Header.Get("Content-Type"))
	filePath := filepath.Join(assetsDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("file creation error: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("file copy error: %w", err)
	}

	return fileName, nil
}

func (ap *AssetProcessor) generateFileName(assetURL, contentType string) string {
	parsedURL, err := url.Parse(assetURL)
	if err == nil && parsedURL.Path != "" {
		fileName := filepath.Base(parsedURL.Path)
		if fileName != "" && fileName != "." {
			return fileName
		}
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(assetURL)))

	ext := ""
	switch {
	case strings.Contains(contentType, "image/png"):
		ext = ".png"
	case strings.Contains(contentType, "image/jpeg"):
		ext = ".jpg"
	case strings.Contains(contentType, "image/gif"):
		ext = ".gif"
	case strings.Contains(contentType, "image/svg+xml"):
		ext = ".svg"
	case strings.Contains(contentType, "text/css"):
		ext = ".css"
	case strings.Contains(contentType, "application/javascript"):
		ext = ".js"
	default:
		ext = ".bin"
	}

	return hash + ext
}
