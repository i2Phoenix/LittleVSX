package extensions

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"littlevsx/internal/config"
	"littlevsx/internal/database"
	"littlevsx/internal/models"
)

const (
	packageJSONPath    = "extension/package.json"
	packageNLSPath     = "extension/package.nls.json"
	maxExtensionsLimit = 10000
	maxSearchLimit     = 1000
	maxQueryLimit      = 100
)

var readmePaths = []string{
	"extension/README.md",
	"extension/readme.md",
	"extension/README",
	"extension/readme",
	"README.md",
	"readme.md",
	"README",
	"readme",
}

type Manager struct {
	directory string
	db        *database.Database
}

func New() (*Manager, error) {
	config := config.GetConfig()
	db, err := database.New()
	if err != nil {
		return nil, err
	}
	return &Manager{
		directory: config.ExtensionsDir,
		db:        db,
	}, nil
}

func (m *Manager) ReadExtensionInfo(filePath string) (*models.Extension, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .vsix file: %w", err)
	}
	defer reader.Close()

	packageJSON, err := m.readPackageJSON(reader)
	if err != nil {
		return nil, err
	}

	pkg, err := m.parsePackageJSON(packageJSON)
	if err != nil {
		return nil, err
	}

	m.processLocalization(reader, pkg)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return m.createExtension(pkg, filePath, fileInfo), nil
}

func (m *Manager) readPackageJSON(reader *zip.ReadCloser) ([]byte, error) {
	for _, file := range reader.File {
		if file.Name == packageJSONPath {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open package.json: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("package.json not found in .vsix file")
}

func (m *Manager) parsePackageJSON(packageJSON []byte) (*packageInfo, error) {
	var pkg packageInfo
	if err := json.Unmarshal(packageJSON, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}
	return &pkg, nil
}

type packageInfo struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Publisher   string         `json:"publisher"`
	Engines     models.Engines `json:"engines"`
	Categories  []string       `json:"categories"`
	Keywords    []string       `json:"keywords"`
	Icon        string         `json:"icon"`
	Repository  interface{}    `json:"repository"`
	Homepage    string         `json:"homepage"`
	Bugs        interface{}    `json:"bugs"`
	License     string         `json:"license"`
}

func (m *Manager) processLocalization(reader *zip.ReadCloser, pkg *packageInfo) {
	if !strings.Contains(pkg.DisplayName, "%") && !strings.Contains(pkg.Description, "%") {
		return
	}

	nlsData := m.readNLSData(reader)
	if nlsData == nil {
		return
	}

	m.replaceLocalizedStrings(pkg, nlsData)
}

func (m *Manager) readNLSData(reader *zip.ReadCloser) map[string]string {
	for _, file := range reader.File {
		if file.Name == packageNLSPath {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			nlsBytes, err := io.ReadAll(rc)
			if err != nil {
				continue
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(nlsBytes, &raw); err != nil {
				continue
			}

			nls := make(map[string]string)
			for key, value := range raw {
				switch v := value.(type) {
				case string:
					nls[key] = v
				case map[string]interface{}:
					if msg, ok := v["message"].(string); ok {
						nls[key] = msg
					}
				}
			}
			return nls
		}
	}
	return nil
}

func (m *Manager) replaceLocalizedStrings(pkg *packageInfo, nls map[string]string) {
	if key := strings.Trim(pkg.DisplayName, "%"); nls[key] != "" {
		pkg.DisplayName = nls[key]
	}
	if key := strings.Trim(pkg.Description, "%"); nls[key] != "" {
		pkg.Description = nls[key]
	}
}

func (m *Manager) createExtension(pkg *packageInfo, filePath string, fileInfo os.FileInfo) *models.Extension {
	extID := fmt.Sprintf("%s.%s", pkg.Publisher, pkg.Name)
	return &models.Extension{
		ID:               extID,
		Name:             pkg.Name,
		DisplayName:      pkg.DisplayName,
		Description:      pkg.Description,
		Version:          pkg.Version,
		Publisher:        pkg.Publisher,
		Engines:          pkg.Engines,
		Categories:       pkg.Categories,
		Tags:             pkg.Keywords,
		Icon:             pkg.Icon,
		Repository:       m.extractRepository(pkg.Repository),
		Homepage:         pkg.Homepage,
		Bugs:             m.extractBugs(pkg.Bugs),
		License:          pkg.License,
		FileSize:         fileInfo.Size(),
		LastUpdated:      fileInfo.ModTime(),
		FilePath:         filePath,
		Verified:         true,
		AverageRating:    5.0,
		ReviewCount:      100,
		DownloadCount:    1000,
		Namespace:        pkg.Publisher,
		ExtensionID:      extID,
		ShortDescription: pkg.Description,
		PublishedDate:    fileInfo.ModTime(),
		ReleaseDate:      fileInfo.ModTime(),
		PreRelease:       false,
		Deprecated:       false,
		TargetPlatform:   "universal",
		ReadmeContent:    m.readReadmeFromVSIX(filePath),
	}
}

func (m *Manager) extractRepository(repo interface{}) string {
	switch v := repo.(type) {
	case string:
		return v
	case map[string]interface{}:
		if url, ok := v["url"].(string); ok {
			return url
		}
	}
	return ""
}

func (m *Manager) extractBugs(bugs interface{}) string {
	switch v := bugs.(type) {
	case string:
		return v
	case map[string]interface{}:
		if url, ok := v["url"].(string); ok {
			return url
		}
	}
	return ""
}

func (m *Manager) GetAll() []*models.Extension {
	extensions, _, err := m.db.GetAllExtensions(1, maxExtensionsLimit)
	if err != nil {
		return []*models.Extension{}
	}
	return database.ToExtensionSlice(extensions)
}

func (m *Manager) GetByID(id string) (*models.Extension, bool) {
	dbExt, err := m.db.GetExtensionByID(id)
	if err != nil {
		return nil, false
	}
	return database.ToExtension(dbExt), true
}

func (m *Manager) Search(query string) []*models.Extension {
	extensions, _, err := m.db.SearchExtensions(query, 1, maxSearchLimit)
	if err != nil {
		return []*models.Extension{}
	}
	return database.ToExtensionSlice(extensions)
}

func (m *Manager) GetFile(id string) (string, bool) {
	dbExt, err := m.db.GetExtensionByID(id)
	if err != nil {
		return "", false
	}
	return dbExt.FilePath, true
}

func (m *Manager) GetStats() map[string]interface{} {
	stats, err := m.db.GetStats()
	if err != nil {
		return map[string]interface{}{
			"total_extensions": 0,
			"publishers":       map[string]int64{},
			"categories":       map[string]int64{},
		}
	}
	return stats
}

func (m *Manager) GetByNamespace(namespace string) []*models.Extension {
	extensions, _, err := m.db.GetExtensionsByPublisher(namespace, 1, maxSearchLimit)
	if err != nil {
		return []*models.Extension{}
	}
	return database.ToExtensionSlice(extensions)
}

func (m *Manager) GetByNamespaceAndName(namespace, name string) (*models.Extension, bool) {
	extID := fmt.Sprintf("%s.%s", namespace, name)
	return m.GetByID(extID)
}

func (m *Manager) GetVersionReferences(namespace, name string) []models.VersionReference {
	ext, ok := m.GetByNamespaceAndName(namespace, name)
	if !ok {
		return nil
	}
	return []models.VersionReference{
		{
			Version:        ext.Version,
			TargetPlatform: ext.TargetPlatform,
			Engines: map[string]string{
				"vscode": ext.Engines.VSCode,
			},
			URL: fmt.Sprintf("/api/-/item/%s/%s/%s", namespace, name, ext.Version),
			Files: map[string]string{
				"download": fmt.Sprintf("/api/extensions/%s/download", ext.ID),
				"manifest": fmt.Sprintf("/api/-/item/%s/%s/%s/file/package.json", namespace, name, ext.Version),
				"icon":     fmt.Sprintf("/api/-/item/%s/%s/%s/file/%s", namespace, name, ext.Version, ext.Icon),
			},
		},
	}
}

func (m *Manager) QueryExtensions(params map[string]string) models.QueryResult {
	all := m.GetAll()
	filtered := m.filterExtensions(all, params)
	offset, size := m.getPaginationParams(params)
	paged := m.applyPagination(filtered, offset, size)
	return models.QueryResult{
		Offset:     offset,
		TotalSize:  len(filtered),
		Extensions: m.toExtensionSlice(paged),
	}
}

func (m *Manager) filterExtensions(exts []*models.Extension, params map[string]string) []*models.Extension {
	var filtered []*models.Extension
	for _, ext := range exts {
		if m.matchesQuery(ext, params) {
			filtered = append(filtered, ext)
		}
	}
	return filtered
}

func (m *Manager) getPaginationParams(params map[string]string) (int, int) {
	offset, size := 0, maxQueryLimit
	fmt.Sscanf(params["offset"], "%d", &offset)
	fmt.Sscanf(params["size"], "%d", &size)
	if size > maxQueryLimit {
		size = maxQueryLimit
	}
	return offset, size
}

func (m *Manager) applyPagination(exts []*models.Extension, offset, size int) []*models.Extension {
	if offset >= len(exts) {
		return []*models.Extension{}
	}
	end := offset + size
	if end > len(exts) {
		end = len(exts)
	}
	return exts[offset:end]
}

func (m *Manager) matchesQuery(ext *models.Extension, params map[string]string) bool {
	if val := params["namespaceName"]; val != "" && ext.Namespace != val {
		return false
	}
	if val := params["extensionName"]; val != "" && ext.Name != val {
		return false
	}
	if val := params["extensionVersion"]; val != "" && ext.Version != val {
		return false
	}
	if val := params["extensionId"]; val != "" && ext.ExtensionID != val {
		return false
	}
	if val := params["targetPlatform"]; val != "" && ext.TargetPlatform != val && ext.TargetPlatform != "universal" {
		return false
	}
	return true
}

func (m *Manager) toExtensionSlice(extensions []*models.Extension) []models.Extension {
	result := make([]models.Extension, len(extensions))
	for i, ext := range extensions {
		result[i] = *ext
	}
	return result
}

func (m *Manager) GetExtensionsDir() string {
	return m.directory
}

func (m *Manager) GetDB() *database.Database {
	return m.db
}

func (m *Manager) readReadmeFromVSIX(filePath string) string {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return ""
	}
	defer reader.Close()

	for _, file := range reader.File {
		for _, path := range readmePaths {
			if file.Name == path || m.isReadmeFile(file.Name) {
				return m.readFileContent(file)
			}
		}
	}
	return ""
}

func (m *Manager) isReadmeFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "readme") &&
		(strings.HasSuffix(lower, ".md") ||
			strings.HasSuffix(lower, ".txt") ||
			!strings.Contains(name, "."))
}

func (m *Manager) readFileContent(file *zip.File) string {
	rc, err := file.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	return string(content)
}

func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *Manager) DeleteExtension(id string) error {
	ext, ok := m.GetByID(id)
	if !ok {
		return fmt.Errorf("extension with ID %s not found", id)
	}

	if err := m.deleteVSIXFile(ext.FilePath); err != nil {
		return fmt.Errorf("failed to delete .vsix file: %w", err)
	}

	if err := m.deleteAssetsFolder(ext.ID); err != nil {
		return fmt.Errorf("failed to delete asset folder: %w", err)
	}

	if err := m.db.DeleteExtension(id); err != nil {
		return fmt.Errorf("failed to delete from database: %w", err)
	}

	return nil
}

func (m *Manager) deleteVSIXFile(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func (m *Manager) deleteAssetsFolder(extensionID string) error {
	config := config.GetConfig()
	assetPath := filepath.Join(config.AssetsDir, extensionID)
	if _, err := os.Stat(assetPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(assetPath)
}
