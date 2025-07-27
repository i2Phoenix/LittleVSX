package utils

import (
	"log"
	"net/http"
	"time"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) LogRequest(r *http.Request) {
	userAgent := l.getHeaderValue(r, "User-Agent", "Unknown")
	referer := l.getHeaderValue(r, "Referer", "Direct")
	accept := l.getHeaderValue(r, "Accept", "Any")

	log.Printf("API Request: %s %s - User-Agent: %s - Referer: %s - Accept: %s",
		r.Method, r.URL.Path, userAgent, referer, accept)

	l.logVSCodiumHeaders(r)

	if accept != "Any" && accept != "*/*" {
		log.Printf("API Version: %s", accept)
	}
}

func (l *Logger) LogResponse(r *http.Request, start time.Time) {
	duration := time.Since(start)
	log.Printf("API Response: %s %s - %v", r.Method, r.URL.Path, duration)
}

func (l *Logger) LogCORS(r *http.Request) {
	log.Printf("API: OPTIONS %s - CORS preflight request", r.URL.Path)
}

func (l *Logger) LogError(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}

func (l *Logger) LogWarning(format string, args ...interface{}) {
	log.Printf("WARNING: "+format, args...)
}

func (l *Logger) LogInfo(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

func (l *Logger) LogDebug(format string, args ...interface{}) {
	log.Printf("DEBUG: "+format, args...)
}

func (l *Logger) LogExtensionInfo(extensionID, displayName, publisher string) {
	log.Printf("Extension Info: ID=%s, Name=%s, Publisher=%s", extensionID, displayName, publisher)
}

func (l *Logger) LogSearchQuery(query string, resultCount int) {
	log.Printf("Search Query: '%s' - found %d results", query, resultCount)
}

func (l *Logger) LogDownloadRequest(extensionID, fileName string) {
	log.Printf("Download Request: %s - file: %s", extensionID, fileName)
}

func (l *Logger) LogStatsRequest() {
	log.Printf("Stats Request")
}

func (l *Logger) LogNotFound(method, path string) {
	log.Printf("API: 404 - Not Found: %s %s", method, path)
}

func (l *Logger) LogMethodNotAllowed(method, path string) {
	log.Printf("API: 405 - Method Not Allowed: %s %s", method, path)
}

func (l *Logger) LogJSONError(err error) {
	log.Printf("Error encoding JSON response: %v", err)
}

func (l *Logger) getHeaderValue(r *http.Request, key, fallback string) string {
	if value := r.Header.Get(key); value != "" {
		return value
	}
	return fallback
}

func (l *Logger) logVSCodiumHeaders(r *http.Request) {
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

func (l *Logger) LogExtensionProcessing(filePath string, stage string) {
	log.Printf("Processing extension: %s - %s", filePath, stage)
}

func (l *Logger) LogDatabaseOperation(operation string, err error) {
	if err != nil {
		log.Printf("Database %s ERROR: %v", operation, err)
	} else {
		log.Printf("Database %s: SUCCESS", operation)
	}
}

func (l *Logger) LogFileOperation(operation, filePath string, err error) {
	if err != nil {
		log.Printf("File %s ERROR: %s - %v", operation, filePath, err)
	} else {
		log.Printf("File %s SUCCESS: %s", operation, filePath)
	}
}

func (l *Logger) LogServerStart(addr string, useHTTPS bool) {
	protocol := "HTTP"
	if useHTTPS {
		protocol = "HTTPS"
	}
	log.Printf("Starting %s server on %s", protocol, addr)
}

func (l *Logger) LogServerStop(err error) {
	if err != nil {
		log.Printf("Server stopped with error: %v", err)
	} else {
		log.Printf("Server stopped gracefully")
	}
}

func (l *Logger) LogConfiguration(config map[string]interface{}) {
	log.Printf("Configuration loaded:")
	for key, value := range config {
		log.Printf("  %s: %v", key, value)
	}
}

func (l *Logger) LogPerformance(operation string, duration time.Duration) {
	if duration > time.Second {
		log.Printf("PERFORMANCE: %s took %v (slow)", operation, duration)
	} else {
		log.Printf("PERFORMANCE: %s took %v", operation, duration)
	}
}

func (l *Logger) LogMemoryUsage(operation string, bytes int64) {
	if bytes > 1024*1024 {
		log.Printf("MEMORY: %s used %d bytes (%.2f MB)", operation, bytes, float64(bytes)/1024/1024)
	} else if bytes > 1024 {
		log.Printf("MEMORY: %s used %d bytes (%.2f KB)", operation, bytes, float64(bytes)/1024)
	} else {
		log.Printf("MEMORY: %s used %d bytes", operation, bytes)
	}
}
