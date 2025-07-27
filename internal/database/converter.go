package database

import (
	"encoding/json"

	"littlevsx/internal/models"
)

func ToDBExtension(ext *models.Extension) *ExtensionDB {
	enginesJSON, _ := json.Marshal(ext.Engines)
	categoriesJSON, _ := json.Marshal(ext.Categories)
	tagsJSON, _ := json.Marshal(ext.Tags)

	return &ExtensionDB{
		ID:               ext.ID,
		Name:             ext.Name,
		DisplayName:      ext.DisplayName,
		Description:      ext.Description,
		Version:          ext.Version,
		Publisher:        ext.Publisher,
		Engines:          string(enginesJSON),
		Categories:       string(categoriesJSON),
		Tags:             string(tagsJSON),
		Icon:             ext.Icon,
		Repository:       ext.Repository,
		Homepage:         ext.Homepage,
		Bugs:             ext.Bugs,
		License:          ext.License,
		FileSize:         ext.FileSize,
		LastUpdated:      ext.LastUpdated,
		FilePath:         ext.FilePath,
		Verified:         ext.Verified,
		AverageRating:    ext.AverageRating,
		ReviewCount:      ext.ReviewCount,
		DownloadCount:    ext.DownloadCount,
		Namespace:        ext.Namespace,
		ExtensionID:      ext.ExtensionID,
		ShortDescription: ext.ShortDescription,
		PublishedDate:    ext.PublishedDate,
		ReleaseDate:      ext.ReleaseDate,
		PreRelease:       ext.PreRelease,
		Deprecated:       ext.Deprecated,
		TargetPlatform:   ext.TargetPlatform,
		ReadmeContent:    ext.ReadmeContent,
	}
}

func ToExtension(dbExt *ExtensionDB) *models.Extension {
	var engines models.Engines
	json.Unmarshal([]byte(dbExt.Engines), &engines)

	var categories []string
	json.Unmarshal([]byte(dbExt.Categories), &categories)

	var tags []string
	json.Unmarshal([]byte(dbExt.Tags), &tags)

	return &models.Extension{
		ID:               dbExt.ID,
		Name:             dbExt.Name,
		DisplayName:      dbExt.DisplayName,
		Description:      dbExt.Description,
		Version:          dbExt.Version,
		Publisher:        dbExt.Publisher,
		Engines:          engines,
		Categories:       categories,
		Tags:             tags,
		Icon:             dbExt.Icon,
		Repository:       dbExt.Repository,
		Homepage:         dbExt.Homepage,
		Bugs:             dbExt.Bugs,
		License:          dbExt.License,
		FileSize:         dbExt.FileSize,
		LastUpdated:      dbExt.LastUpdated,
		FilePath:         dbExt.FilePath,
		Verified:         dbExt.Verified,
		AverageRating:    dbExt.AverageRating,
		ReviewCount:      dbExt.ReviewCount,
		DownloadCount:    dbExt.DownloadCount,
		Namespace:        dbExt.Namespace,
		ExtensionID:      dbExt.ExtensionID,
		ShortDescription: dbExt.ShortDescription,
		PublishedDate:    dbExt.PublishedDate,
		ReleaseDate:      dbExt.ReleaseDate,
		PreRelease:       dbExt.PreRelease,
		Deprecated:       dbExt.Deprecated,
		TargetPlatform:   dbExt.TargetPlatform,
		ReadmeContent:    dbExt.ReadmeContent,
	}
}

func ToExtensionSlice(dbExtensions []ExtensionDB) []*models.Extension {
	result := make([]*models.Extension, len(dbExtensions))
	for i, dbExt := range dbExtensions {
		result[i] = ToExtension(&dbExt)
	}
	return result
}
