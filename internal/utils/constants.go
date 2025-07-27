package utils

const (
	ContentTypeHeader        = "Content-Type"
	ContentDispositionHeader = "Content-Disposition"
	CacheControlHeader       = "Cache-Control"
)

const (
	JSONContentType        = "application/json"
	XMLContentType         = "application/xml"
	MarkdownContentType    = "text/markdown"
	OctetStreamContentType = "application/octet-stream"
)

const (
	CacheControlValue = "public, max-age=31536000"
)

const (
	MaxSearchResults = 1000
	MaxQuerySize     = 100
	MaxPageSize      = 100
)

const (
	PackageJSONPath  = "extension/package.json"
	VSIXManifestPath = "extension.vsixmanifest"
	ReadmePath       = "extension/README.md"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
)

const (
	CORSAllowOrigin      = "*"
	CORSAllowMethods     = "OPTIONS,GET,POST,PATCH,PUT,DELETE"
	CORSAllowHeaders     = "Content-Type,Authorization,Accept,X-Requested-With,X-Market-Client-Id,X-Market-User-Id,X-Client-Commit,X-Client-Name,X-Client-Version,X-Machine-Id,VSCode-SessionId,accept"
	CORSAllowCredentials = "true"
	CORSMaxAge           = "86400"
)

const (
	HTTPAPIVersion         = "application/json;api-version=3.0-preview.1"
	HTTPCacheControl       = "no-cache, no-store, max-age=0, must-revalidate"
	HTTPPragma             = "no-cache"
	HTTPExpires            = "0"
	HTTPContentTypeOptions = "nosniff"
	HTTPXSSProtection      = "0"
	HTTPFrameOptions       = "DENY"
	HTTPHSTS               = "max-age=31536000 ; includeSubDomains"
)
