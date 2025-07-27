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

type ExtensionInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	DownloadURL string `json:"downloadUrl"`
	FileSize    int64  `json:"fileSize"`
}

type Marketplace struct {
	client *http.Client
}

func New() *Marketplace {
	return &Marketplace{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (m *Marketplace) GetExtensionInfo(marketplaceURL string) (*ExtensionInfo, error) {
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

func (m *Marketplace) GetExtensionInfoByID(extensionID string) (*ExtensionInfo, error) {
	return m.fetchExtensionInfo(extensionID)
}

type DownloadResult struct {
	FilePath      string
	WasDownloaded bool
}

func (m *Marketplace) DownloadExtension(info *ExtensionInfo, targetDir string) (*DownloadResult, error) {
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

func (m *Marketplace) extractExtensionID(parsedURL *url.URL) (string, error) {
	if itemName := parsedURL.Query().Get("itemName"); itemName != "" {
		return itemName, nil
	}

	patterns := []string{
		`/items/([^/]+/[^/]+)`,
		`/extension/([^/]+/[^/]+)`,
		`/marketplace/item/([^/]+/[^/]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(parsedURL.Path); len(matches) > 1 {
			return matches[1], nil
		}
	}

	fullURL := parsedURL.String()
	itemNamePattern := regexp.MustCompile(`[?&]itemName=([^&]+)`)
	if matches := itemNamePattern.FindStringSubmatch(fullURL); len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("could not extract extension ID from URL: %s", parsedURL.String())
}

func (m *Marketplace) fetchExtensionInfo(extensionID string) (*ExtensionInfo, error) {
	apiURL := "https://marketplace.visualstudio.com/_apis/public/gallery/extensionquery"

	requestBody := map[string]interface{}{
		"filters": []map[string]interface{}{
			{
				"criteria": []map[string]interface{}{
					{
						"filterType": 7,
						"value":      extensionID,
					},
				},
				"pageNumber": 1,
				"pageSize":   1,
			},
		},
		"flags": 2151,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json; api-version=3.0-preview.1")

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
		Results []struct {
			Extensions []struct {
				ExtensionID      string `json:"extensionId"`
				ExtensionName    string `json:"extensionName"`
				DisplayName      string `json:"displayName"`
				ShortDescription string `json:"shortDescription"`
				Versions         []struct {
					Version string `json:"version"`
					Files   []struct {
						AssetType string `json:"assetType"`
						Source    string `json:"source"`
					} `json:"files"`
				} `json:"versions"`
				Publisher struct {
					PublisherName string `json:"publisherName"`
				} `json:"publisher"`
			} `json:"extensions"`
		} `json:"results"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Results) == 0 || len(response.Results[0].Extensions) == 0 {
		return nil, fmt.Errorf("extension not found: %s", extensionID)
	}

	ext := response.Results[0].Extensions[0]

	if len(ext.Versions) == 0 {
		return nil, fmt.Errorf("no versions found for extension")
	}

	latestVersion := ext.Versions[0]
	var downloadURL string

	for _, file := range latestVersion.Files {
		if file.AssetType == "Microsoft.VisualStudio.Services.VSIXPackage" {
			downloadURL = file.Source
			break
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("download URL not found")
	}

	return &ExtensionInfo{
		ID:          ext.ExtensionID,
		Name:        ext.ExtensionName,
		DisplayName: ext.DisplayName,
		Description: ext.ShortDescription,
		Version:     latestVersion.Version,
		Publisher:   ext.Publisher.PublisherName,
		DownloadURL: downloadURL,
	}, nil
}

func (m *Marketplace) downloadFile(downloadURL, filePath string) error {
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
