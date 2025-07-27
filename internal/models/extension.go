package models

import (
	"time"
)

type Extension struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"displayName"`
	Description      string    `json:"description"`
	Version          string    `json:"version"`
	Publisher        string    `json:"publisher"`
	Engines          Engines   `json:"engines"`
	Categories       []string  `json:"categories,omitempty"`
	Tags             []string  `json:"tags,omitempty"`
	Icon             string    `json:"icon,omitempty"`
	Repository       string    `json:"repository,omitempty"`
	Homepage         string    `json:"homepage,omitempty"`
	Bugs             string    `json:"bugs,omitempty"`
	License          string    `json:"license,omitempty"`
	FileSize         int64     `json:"fileSize"`
	LastUpdated      time.Time `json:"lastUpdated"`
	FilePath         string    `json:"filePath"`
	Verified         bool      `json:"verified"`
	AverageRating    float64   `json:"averageRating"`
	ReviewCount      int64     `json:"reviewCount"`
	DownloadCount    int64     `json:"downloadCount"`
	Namespace        string    `json:"namespace"`
	ExtensionID      string    `json:"extensionId"`
	ShortDescription string    `json:"shortDescription"`
	PublishedDate    time.Time `json:"publishedDate"`
	ReleaseDate      time.Time `json:"releaseDate"`
	PreRelease       bool      `json:"preRelease"`
	Deprecated       bool      `json:"deprecated"`
	TargetPlatform   string    `json:"targetPlatform"`
	ReadmeContent    string    `json:"readmeContent"`
}

type Engines struct {
	VSCode string `json:"vscode"`
}

type VersionReference struct {
	Version        string            `json:"version"`
	TargetPlatform string            `json:"targetPlatform"`
	Engines        map[string]string `json:"engines"`
	URL            string            `json:"url"`
	Files          map[string]string `json:"files"`
}

type Namespace struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Website     string    `json:"website"`
	Verified    bool      `json:"verified"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type QueryResult struct {
	Offset     int         `json:"offset"`
	TotalSize  int         `json:"totalSize"`
	Extensions []Extension `json:"extensions"`
}

type SearchResult struct {
	Offset     int         `json:"offset"`
	TotalSize  int         `json:"totalSize"`
	Extensions []Extension `json:"extensions"`
}
