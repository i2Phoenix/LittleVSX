package server

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"littlevsx/internal/extensions"
	"littlevsx/internal/models"
	"littlevsx/internal/utils"

	"github.com/gorilla/mux"
)

const (
	contentTypeHeader        = "Content-Type"
	contentDispositionHeader = "Content-Disposition"
	cacheControlHeader       = "Cache-Control"

	jsonContentType        = "application/json"
	xmlContentType         = "application/xml"
	markdownContentType    = "text/markdown"
	octetStreamContentType = "application/octet-stream"

	packageJSONPath  = "extension/package.json"
	vsixManifestPath = "extension.vsixmanifest"
	readmePaths      = "extension/README.md"
)

type Server struct {
	extManager *extensions.Manager
	router     *mux.Router
	server     *http.Server
	useHTTPS   bool
	certFile   string
	keyFile    string
	baseURL    string
}

func New(extManager *extensions.Manager, baseURL string) *Server {
	s := &Server{
		extManager: extManager,
		router:     mux.NewRouter(),
		useHTTPS:   false,
		baseURL:    baseURL,
	}
	s.setupRoutes()
	return s
}

func NewWithHTTPS(extManager *extensions.Manager, certFile, keyFile string, baseURL string) *Server {
	s := &Server{
		extManager: extManager,
		router:     mux.NewRouter(),
		useHTTPS:   true,
		certFile:   certFile,
		keyFile:    keyFile,
		baseURL:    baseURL,
	}
	s.setupRoutes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	if s.useHTTPS {
		log.Printf("Starting HTTPS server on %s", addr)
		return s.server.ListenAndServeTLS(s.certFile, s.keyFile)
	}
	log.Printf("Starting HTTP server on %s", addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) setupRoutes() {
	root := s.router.PathPrefix("/").Subrouter()

	root.HandleFunc("/", s.handleRoot).Methods("GET", "OPTIONS")

	root.HandleFunc("/_apis/public/gallery/extensionquery", s.handleExtensionQuery).Methods("POST", "OPTIONS")

	root.HandleFunc("/_gallery/{publisher}/{name}/latest", s.handleVSCodeExtension).Methods("GET", "OPTIONS")

	root.HandleFunc("/_assets/{publisher}/{name}/{version}/{assetType}", s.handleVSCodeAsset).Methods("GET", "OPTIONS")
	root.HandleFunc("/_assets/{extensionID}/{filename}", s.handleExtensionAssets).Methods("GET", "OPTIONS")

	s.router.Use(s.corsMiddleware)
	s.router.Use(s.loggingMiddleware)

	s.router.NotFoundHandler = http.HandlerFunc(s.handleNotFound)
	s.router.MethodNotAllowedHandler = http.HandlerFunc(s.handleMethodNotAllowed)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		s.logRequest(r)

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		log.Printf("API Response: %s %s - %v", r.Method, r.URL.Path, duration)
	})
}

func (s *Server) logRequest(r *http.Request) {
	userAgent := s.getHeaderValue(r, "User-Agent", "Unknown")
	referer := s.getHeaderValue(r, "Referer", "Direct")
	accept := s.getHeaderValue(r, "Accept", "Any")

	log.Printf("API Request: %s %s - User-Agent: %s - Referer: %s - Accept: %s",
		r.Method, r.URL.Path, userAgent, referer, accept)

	s.logVSCodiumHeaders(r)

	if accept != "Any" && accept != "*/*" {
		log.Printf("API Version: %s", accept)
	}
}

func (s *Server) getHeaderValue(r *http.Request, key, fallback string) string {
	if value := r.Header.Get(key); value != "" {
		return value
	}
	return fallback
}

func (s *Server) logVSCodiumHeaders(r *http.Request) {
	headers := []string{
		r.Header.Get("X-Market-Client-Id"),
		r.Header.Get("X-Market-User-Id"),
		r.Header.Get("X-Client-Name"),
		r.Header.Get("X-Client-Version"),
	}

	hasHeaders := false
	for _, header := range headers {
		if header != "" {
			hasHeaders = true
			break
		}
	}

	if hasHeaders {
		log.Printf("API VSCodium Headers: Client-Id: %s, User-Id: %s, Client: %s, Version: %s",
			headers[0], headers[1], headers[2], headers[3])
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.setCORSHeaders(w)
		s.setHTTPHeaders(w)

		if r.Method == "OPTIONS" {
			log.Printf("API: OPTIONS %s - CORS preflight request", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PATCH,PUT,DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Accept,X-Requested-With,X-Market-Client-Id,X-Market-User-Id,X-Client-Commit,X-Client-Name,X-Client-Version,X-Machine-Id,VSCode-SessionId,accept")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

func (s *Server) setHTTPHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", utils.HTTPCacheControl)
	w.Header().Set("Pragma", utils.HTTPPragma)
	w.Header().Set("Expires", utils.HTTPExpires)
	w.Header().Set("X-Content-Type-Options", utils.HTTPContentTypeOptions)
	w.Header().Set("X-XSS-Protection", utils.HTTPXSSProtection)
	w.Header().Set("X-Frame-Options", utils.HTTPFrameOptions)
	w.Header().Set("Strict-Transport-Security", utils.HTTPHSTS)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("API: GET / - root endpoint request")

	info := map[string]interface{}{
		"name":        "LittleVSX",
		"description": "Local marketplace for Visual Studio Code",
		"version":     "1.0.0",
		"endpoints": map[string]string{
			"vscode": "/_apis/public/gallery/extensionquery",
		},
	}

	s.writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleExtensionQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var query map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		log.Printf("API: POST %s - invalid JSON body: %v", r.URL.Path, err)
		s.writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	log.Printf("API: POST %s - received query: %+v", r.URL.Path, query)

	var searchQuery string
	var extensionId string

	if q, ok := query["query"].(string); ok && q != "" {
		searchQuery = q
	} else if filters, ok := query["filters"].([]interface{}); ok && len(filters) > 0 {
		if filter, ok := filters[0].(map[string]interface{}); ok {
			if criteria, ok := filter["criteria"].([]interface{}); ok && len(criteria) > 0 {
				for _, criterion := range criteria {
					if criterionMap, ok := criterion.(map[string]interface{}); ok {
						if filterType, ok := criterionMap["filterType"].(float64); ok {
							switch filterType {
							case 10: // filterType 10 = Search query
								if value, ok := criterionMap["value"].(string); ok {
									searchQuery = value
								}
							case 4: // filterType 4 = Extension ID
								if value, ok := criterionMap["value"].(string); ok {
									extensionId = value
								}
							}
						}
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", utils.HTTPAPIVersion)

	var results []interface{}

	if extensionId != "" {
		log.Printf("API: POST %s - searching by extension ID: '%s'", r.URL.Path, extensionId)
		ext, found := s.extManager.GetByID(extensionId)
		if found && ext != nil {
			extensionInfo := s.createExtensionInfo(ext)
			if extensionInfo != nil {
				results = []interface{}{extensionInfo}
			}
		}
	} else if searchQuery != "" {
		log.Printf("API: POST %s - search query: '%s'", r.URL.Path, searchQuery)
		extensions := s.extManager.Search(searchQuery)
		for _, ext := range extensions {
			if ext != nil {
				extensionInfo := s.createExtensionInfo(ext)
				if extensionInfo != nil {
					results = append(results, extensionInfo)
				}
			}
		}
	} else {
		log.Printf("API: POST %s - no search query or extension ID found, returning all extensions", r.URL.Path)
		allExtensions := s.extManager.GetAll()
		for _, ext := range allExtensions {
			if ext != nil {
				extensionInfo := s.createExtensionInfo(ext)
				if extensionInfo != nil {
					results = append(results, extensionInfo)
				}
			}
		}
	}

	if results == nil {
		results = []interface{}{}
	}

	response := map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"extensions": results,
				"resultMetadata": []map[string]interface{}{
					{
						"metadataType": "ResultCount",
						"metadataItems": []map[string]interface{}{
							{
								"name":  "TotalCount",
								"count": len(results),
							},
						},
					},
				},
			},
		},
	}

	log.Printf("API: POST %s - returning %d results", r.URL.Path, len(results))
	if len(results) == 0 {
		log.Printf("API: POST %s - no results found, returning empty array", r.URL.Path)
	}
	log.Printf("API: POST %s - response structure: %+v", r.URL.Path, response)
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) createExtensionInfo(ext *models.Extension) map[string]interface{} {
	extensionId := ext.ID
	if extensionId == "" {
		extensionId = generateUUID()
	}

	// Создаем версию расширения
	version := map[string]interface{}{
		"version":          ext.Version,
		"lastUpdated":      ext.LastUpdated,
		"assetUri":         fmt.Sprintf("%s/_assets/%s/%s/%s", s.baseURL, ext.Publisher, ext.Name, ext.Version),
		"fallbackAssetUri": fmt.Sprintf("%s/_assets/%s/%s/%s", s.baseURL, ext.Publisher, ext.Name, ext.Version),
		"targetPlatform":   "universal",
		"files": []map[string]interface{}{
			{
				"assetType": "Microsoft.VisualStudio.Code.Manifest",
				"source":    fmt.Sprintf("%s/_gallery/%s/%s/%s/file/package.json", s.baseURL, ext.Publisher, ext.Name, ext.Version),
			},
			{
				"assetType": "Microsoft.VisualStudio.Services.VSIXPackage",
				"source":    fmt.Sprintf("%s/_gallery/%s/%s/%s/file/%s", s.baseURL, ext.Publisher, ext.Name, ext.Version, filepath.Base(ext.FilePath)),
			},
			{
				"assetType": "Microsoft.VisualStudio.Services.VsixManifest",
				"source":    fmt.Sprintf("%s/_gallery/%s/%s/%s/file/extension.vsixmanifest", s.baseURL, ext.Publisher, ext.Name, ext.Version),
			},
			{
				"assetType": "Microsoft.VisualStudio.Services.VsixSignature",
				"source":    fmt.Sprintf("%s/_gallery/%s/%s/%s/file/%s.sigzip", s.baseURL, ext.Publisher, ext.Name, ext.Version, strings.TrimSuffix(filepath.Base(ext.FilePath), ".vsix")),
			},
			{
				"assetType": "Microsoft.VisualStudio.Services.PublicKey",
				"source":    fmt.Sprintf("%s/_gallery/-/public-key/%s", s.baseURL, generateUUID()),
			},
		},
		"properties": []map[string]interface{}{
			{"key": "Microsoft.VisualStudio.Services.Branding.Color", "value": ""},
			{"key": "Microsoft.VisualStudio.Services.Branding.Theme", "value": ""},
			{"key": "Microsoft.VisualStudio.Services.Links.Source", "value": ext.Repository},
			{"key": "Microsoft.VisualStudio.Code.SponsorLink", "value": ""},
			{"key": "Microsoft.VisualStudio.Code.Engine", "value": ext.Engines.VSCode},
			{"key": "Microsoft.VisualStudio.Code.ExtensionDependencies", "value": ""},
			{"key": "Microsoft.VisualStudio.Code.ExtensionPack", "value": ""},
			{"key": "Microsoft.VisualStudio.Code.LocalizedLanguages", "value": ""},
			{"key": "Microsoft.VisualStudio.Code.PreRelease", "value": "false"},
		},
	}

	// Добавляем README если есть
	if ext.ReadmeContent != "" || ext.Description != "" {
		version["files"] = append(version["files"].([]map[string]interface{}), map[string]interface{}{
			"assetType": "Microsoft.VisualStudio.Services.Content.Details",
			"source":    fmt.Sprintf("%s/_assets/%s/%s/%s/Microsoft.VisualStudio.Services.Content.Details", s.baseURL, ext.Publisher, ext.Name, ext.Version),
		})
	}

	// Добавляем LICENSE если есть
	if ext.License != "" {
		version["files"] = append(version["files"].([]map[string]interface{}), map[string]interface{}{
			"assetType": "Microsoft.VisualStudio.Services.Content.License",
			"source":    fmt.Sprintf("%s/_assets/%s/%s/%s/file/LICENSE.md", s.baseURL, ext.Publisher, ext.Name, ext.Version),
		})
	}

	// Добавляем иконку если есть
	if ext.Icon != "" {
		version["files"] = append(version["files"].([]map[string]interface{}), map[string]interface{}{
			"assetType": "Microsoft.VisualStudio.Services.Icons.Default",
			"source":    fmt.Sprintf("%s/_assets/%s/%s/%s/file/%s", s.baseURL, ext.Publisher, ext.Name, ext.Version, filepath.Base(ext.Icon)),
		})
	}

	return map[string]interface{}{
		"extensionId":      extensionId,
		"extensionName":    ext.Name,
		"displayName":      ext.DisplayName,
		"shortDescription": ext.Description,
		"publisher": map[string]interface{}{
			"displayName":      ext.Publisher,
			"publisherId":      generateUUID(),
			"publisherName":    ext.Publisher,
			"domain":           nil,
			"isDomainVerified": nil,
		},
		"versions": []map[string]interface{}{version},
		"statistics": []map[string]interface{}{
			{"statisticName": "install", "value": 0.0},
			{"statisticName": "ratingcount", "value": 0.0},
		},
		"tags":          ext.Tags,
		"releaseDate":   ext.LastUpdated,
		"publishedDate": ext.LastUpdated,
		"lastUpdated":   ext.LastUpdated,
		"categories":    ext.Categories,
		"flags":         "",
	}
}

func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *Server) handleVSCodeExtension(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	publisher := vars["publisher"]
	name := vars["name"]

	extensionID := fmt.Sprintf("%s.%s", publisher, name)

	log.Printf("API: GET /_gallery/%s/%s/latest - looking for extension: %s", publisher, name, extensionID)

	ext, exists := s.extManager.GetByID(extensionID)
	if !exists {
		log.Printf("API: GET /_gallery/%s/%s/latest - NOT FOUND: %s", publisher, name, extensionID)
		s.writeError(w, http.StatusNotFound, "Extension not found")
		return
	}

	log.Printf("API: GET /_gallery/%s/%s/latest - FOUND: %s by %s", publisher, name, ext.DisplayName, ext.Publisher)
	s.writeJSON(w, http.StatusOK, ext)
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("API: 404 - Not Found: %s %s", r.Method, r.URL.Path)
	s.writeError(w, http.StatusNotFound, "Page not found")
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("API: 405 - Method Not Allowed: %s %s", r.Method, r.URL.Path)
	s.writeError(w, http.StatusMethodNotAllowed, "Method not supported")
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	if contentType := w.Header().Get(contentTypeHeader); contentType == "" || !strings.Contains(contentType, "api-version") {
		w.Header().Set(contentTypeHeader, jsonContentType)
	}
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "JSON encoding error", http.StatusInternalServerError)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	errorResponse := map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
		"status":  status,
	}

	s.writeJSON(w, status, errorResponse)
}

func (s *Server) handleVSCodeAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	publisher := vars["publisher"]
	name := vars["name"]
	version := vars["version"]
	assetType := vars["assetType"]

	extensionID := fmt.Sprintf("%s.%s", publisher, name)

	log.Printf("API: GET /_assets/%s/%s/%s/%s - asset request", publisher, name, version, assetType)

	ext, exists := s.extManager.GetByID(extensionID)
	if !exists {
		log.Printf("API: GET /_assets/%s/%s/%s/%s - EXTENSION NOT FOUND", publisher, name, version, assetType)
		s.writeError(w, http.StatusNotFound, "Extension not found")
		return
	}

	if ext.Version != version {
		log.Printf("API: GET /_assets/%s/%s/%s/%s - VERSION NOT FOUND (available: %s)", publisher, name, version, assetType, ext.Version)
		s.writeError(w, http.StatusNotFound, "Version not found")
		return
	}

	switch assetType {
	case "Microsoft.VisualStudio.Code.Manifest":
		s.servePackageJSON(w, ext)
	case "Microsoft.VisualStudio.Services.VSIXPackage":
		s.serveVSIXFile(w, r, ext)
	case "Microsoft.VisualStudio.Services.VsixManifest":
		s.serveVSIXManifest(w, ext)
	case "Microsoft.VisualStudio.Services.VsixSignature":
		s.serveEmptySignature(w)
	case "Microsoft.VisualStudio.Services.PublicKey":
		s.serveEmptyPublicKey(w)
	case "Microsoft.VisualStudio.Services.Content.Details":
		s.serveREADME(w, ext)
	case "Microsoft.VisualStudio.Services.Content.License":
		s.serveLICENSE(w, ext)
	case "Microsoft.VisualStudio.Services.Icons.Default":
		s.serveIcon(w, ext)
	default:
		log.Printf("API: GET /_assets/%s/%s/%s/%s - UNKNOWN ASSET TYPE", publisher, name, version, assetType)
		s.writeError(w, http.StatusNotFound, "Asset type not supported")
	}
}

func (s *Server) servePackageJSON(w http.ResponseWriter, ext *models.Extension) {
	packageJSON, err := s.extractFileFromVSIX(ext.FilePath, packageJSONPath)
	if err != nil {
		log.Printf("API: Error extracting package.json: %v", err)
		w.Header().Set("Content-Type", "application/json")
		basicInfo := map[string]interface{}{
			"name":        ext.Name,
			"displayName": ext.DisplayName,
			"description": ext.Description,
			"version":     ext.Version,
			"publisher":   ext.Publisher,
			"engines":     ext.Engines,
			"categories":  ext.Categories,
			"tags":        ext.Tags,
			"icon":        ext.Icon,
			"repository":  ext.Repository,
			"homepage":    ext.Homepage,
			"bugs":        ext.Bugs,
			"license":     ext.License,
		}
		jsonData, _ := json.Marshal(basicInfo)
		w.Write(jsonData)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(packageJSON)
}

func (s *Server) serveVSIXFile(w http.ResponseWriter, r *http.Request, ext *models.Extension) {
	fileName := filepath.Base(ext.FilePath)
	w.Header().Set(contentDispositionHeader, fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", octetStreamContentType)
	http.ServeFile(w, r, ext.FilePath)
}

func (s *Server) serveVSIXManifest(w http.ResponseWriter, ext *models.Extension) {
	manifest, err := s.extractFileFromVSIX(ext.FilePath, vsixManifestPath)
	if err != nil {
		log.Printf("API: Error extracting extension.vsixmanifest: %v", err)
		w.Header().Set("Content-Type", xmlContentType)
		basicManifest := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011">
  <Metadata>
    <Identity Id="%s" Version="%s" Publisher="%s" Language="en-US" />
    <DisplayName>%s</DisplayName>
    <Description>%s</Description>
  </Metadata>
</PackageManifest>`, ext.ID, ext.Version, ext.Publisher, ext.DisplayName, ext.Description)
		w.Write([]byte(basicManifest))
		return
	}

	w.Header().Set("Content-Type", xmlContentType)
	w.Write(manifest)
}

func (s *Server) serveEmptySignature(w http.ResponseWriter) {
	w.Header().Set("Content-Type", octetStreamContentType)
	w.Write([]byte{})
}

func (s *Server) serveEmptyPublicKey(w http.ResponseWriter) {
	w.Header().Set("Content-Type", octetStreamContentType)
	w.Write([]byte{})
}

func (s *Server) serveREADME(w http.ResponseWriter, ext *models.Extension) {
	w.Header().Set("Content-Type", markdownContentType)

	if ext.ReadmeContent != "" {
		w.Write([]byte(ext.ReadmeContent))
	} else {
		readme, err := s.extractFileFromVSIX(ext.FilePath, readmePaths)
		if err != nil {
			message := fmt.Sprintf("# %s\n\nDescription for this extension is not available.\n\n**Publisher:** %s\n**Version:** %s",
				ext.DisplayName, ext.Publisher, ext.Version)
			w.Write([]byte(message))
			return
		}
		w.Write(readme)
	}
}

func (s *Server) serveLICENSE(w http.ResponseWriter, ext *models.Extension) {
	license, err := s.extractFileFromVSIX(ext.FilePath, "extension/LICENSE.md")
	if err != nil {
		w.Header().Set("Content-Type", markdownContentType)
		message := fmt.Sprintf("# License\n\nLicense information for extension **%s** is not available.\n\n**Publisher:** %s\n**Version:** %s",
			ext.DisplayName, ext.Publisher, ext.Version)
		w.Write([]byte(message))
		return
	}

	w.Header().Set("Content-Type", markdownContentType)
	w.Write(license)
}

func (s *Server) serveIcon(w http.ResponseWriter, ext *models.Extension) {
	if ext.Icon == "" {
		w.Header().Set("Content-Type", "text/plain")
		message := fmt.Sprintf("Icon for extension %s is not available", ext.DisplayName)
		w.Write([]byte(message))
		return
	}

	iconPath := fmt.Sprintf("extension/%s", ext.Icon)
	icon, err := s.extractFileFromVSIX(ext.FilePath, iconPath)
	if err != nil {
		log.Printf("API: Error extracting icon: %v", err)
		w.Header().Set("Content-Type", "text/plain")
		message := fmt.Sprintf("Icon for extension %s not found", ext.DisplayName)
		w.Write([]byte(message))
		return
	}

	fileExt := filepath.Ext(ext.Icon)
	var mimeType string
	switch strings.ToLower(fileExt) {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".svg":
		mimeType = "image/svg+xml"
	default:
		mimeType = "image/png"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Write(icon)
}

func (s *Server) extractFileFromVSIX(vsixPath, filePath string) ([]byte, error) {
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

func (s *Server) handleExtensionAssets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	extensionID := vars["extensionID"]
	filename := vars["filename"]

	if extensionID == "" || filename == "" {
		s.writeError(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	assetsDir := filepath.Join(s.extManager.GetExtensionsDir(), "assets", extensionID)
	filePath := filepath.Join(assetsDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		s.writeError(w, http.StatusNotFound, "Asset not found")
		return
	}

	contentType := "application/octet-stream"
	fileExt := strings.ToLower(filepath.Ext(filename))

	switch fileExt {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".svg":
		contentType = "image/svg+xml; charset=utf-8"
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".ico":
		contentType = "image/x-icon"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = s.detectContentType(filePath)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")

	http.ServeFile(w, r, filePath)
}

func (s *Server) detectContentType(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	bytesRead, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "application/octet-stream"
	}

	contentType := http.DetectContentType(buffer[:bytesRead])

	if strings.Contains(string(buffer[:bytesRead]), "<?xml") ||
		strings.Contains(string(buffer[:bytesRead]), "<svg") {
		return "image/svg+xml; charset=utf-8"
	}

	return contentType
}
