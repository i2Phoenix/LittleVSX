package marketplace

import (
	"encoding/json"
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

type OpenVSXMarketplace struct {
	client *http.Client
}

func NewOpenVSX() *OpenVSXMarketplace {
	return &OpenVSXMarketplace{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (m *OpenVSXMarketplace) GetName() string {
	return "Open VSX Registry"
}

func (m *OpenVSXMarketplace) GetExtensionInfo(marketplaceURL string) (*ExtensionInfo, error) {
	cleanURL := strings.ReplaceAll(marketplaceURL, "\\", "")
	parsedURL, err := url.Parse(cleanURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	extensionID, err := m.extractExtensionID(parsedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract extension ID: %w", err)
	}

	info, err := m.fetchExtensionInfo(extensionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch extension info: %w", err)
	}

	return info, nil
}

func (m *OpenVSXMarketplace) GetExtensionInfoByID(extensionID string) (*ExtensionInfo, error) {
	return m.fetchExtensionInfo(extensionID)
}

func (m *OpenVSXMarketplace) DownloadExtension(info *ExtensionInfo, targetDir string) (*DownloadResult, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fileName := fmt.Sprintf("%s-%s.vsix", info.Name, info.Version)
	filePath := filepath.Join(targetDir, fileName)

	if _, err := os.Stat(filePath); err == nil {
		return &DownloadResult{FilePath: filePath, WasDownloaded: false}, nil
	}

	if err := m.downloadFile(info.DownloadURL, filePath); err != nil {
		return nil, err
	}

	return &DownloadResult{FilePath: filePath, WasDownloaded: true}, nil
}

func (m *OpenVSXMarketplace) extractExtensionID(parsedURL *url.URL) (string, error) {
	// Open VSX Registry URL pattern: /extension/publisher/name
	// Example: /extension/jeanp413/open-remote-ssh
	patterns := []string{
		`/extension/([^/]+/[^/]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(parsedURL.Path); len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not extract extension ID from Open VSX URL: %s", parsedURL.String())
}

func (m *OpenVSXMarketplace) fetchExtensionInfo(extensionID string) (*ExtensionInfo, error) {
	// Open VSX Registry API endpoint
	apiURL := fmt.Sprintf("https://open-vsx.org/api/-/query?extensionId=%s", extensionID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Extensions []struct {
			ExtensionName string `json:"name"`
			DisplayName   string `json:"displayName"`
			Description   string `json:"description"`
			Publisher     string `json:"namespace"`
			LatestVersion string `json:"version"`
			Files         struct {
				Download string `json:"download"`
			} `json:"files"`
		} `json:"extensions"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Extensions) == 0 {
		return nil, fmt.Errorf("extension not found: %s", extensionID)
	}

	ext := response.Extensions[0]

	if ext.Files.Download == "" {
		return nil, fmt.Errorf("download URL not found")
	}

	// Construct the full extension ID from namespace and name
	fullExtensionID := fmt.Sprintf("%s.%s", ext.Publisher, ext.ExtensionName)

	return &ExtensionInfo{
		ID:          fullExtensionID,
		Name:        ext.ExtensionName,
		DisplayName: ext.DisplayName,
		Description: ext.Description,
		Version:     ext.LatestVersion,
		Publisher:   ext.Publisher,
		DownloadURL: ext.Files.Download,
	}, nil
}

func (m *OpenVSXMarketplace) downloadFile(downloadURL, filePath string) error {
	resp, err := m.client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Downloaded: %s (%d bytes)\n", filePath, written)
	return nil
}
