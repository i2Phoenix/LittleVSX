package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"littlevsx/internal/config"

	_ "modernc.org/sqlite"
)

type ExtensionDB struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"displayName"`
	Description      string    `json:"description"`
	Version          string    `json:"version"`
	Publisher        string    `json:"publisher"`
	Engines          string    `json:"engines"`
	Categories       string    `json:"categories"`
	Tags             string    `json:"tags"`
	Icon             string    `json:"icon"`
	Repository       string    `json:"repository"`
	Homepage         string    `json:"homepage"`
	Bugs             string    `json:"bugs"`
	License          string    `json:"license"`
	FileSize         int64     `json:"fileSize"`
	LastUpdated      time.Time `json:"lastUpdated"`
	FilePath         string    `json:"filePath"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
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

type Database struct {
	db *sql.DB
}

func New() (*Database, error) {
	cfg := config.GetConfig()

	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if cfg.AutoMigrate {
		if err := createTables(db); err != nil {
			return nil, fmt.Errorf("database migration error: %w", err)
		}
		log.Println("Database migration completed")
	}

	return &Database{db: db}, nil
}

func createTables(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS extensions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT,
		description TEXT,
		version TEXT NOT NULL,
		publisher TEXT NOT NULL,
		engines TEXT,
		categories TEXT,
		tags TEXT,
		icon TEXT,
		repository TEXT,
		homepage TEXT,
		bugs TEXT,
		license TEXT,
		file_size INTEGER NOT NULL,
		last_updated DATETIME NOT NULL,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		verified BOOLEAN DEFAULT 1,
		average_rating REAL DEFAULT 5.0,
		review_count INTEGER DEFAULT 100,
		download_count INTEGER DEFAULT 1000,
		namespace TEXT,
		extension_id TEXT,
		short_description TEXT,
		published_date DATETIME,
		release_date DATETIME,
		pre_release BOOLEAN DEFAULT 0,
		deprecated BOOLEAN DEFAULT 0,
		target_platform TEXT DEFAULT 'universal',
		readme_content TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_extensions_name ON extensions(name);
	CREATE INDEX IF NOT EXISTS idx_extensions_publisher ON extensions(publisher);
	CREATE INDEX IF NOT EXISTS idx_extensions_file_path ON extensions(file_path);
	CREATE INDEX IF NOT EXISTS idx_extensions_last_updated ON extensions(last_updated);
	`

	_, err := db.Exec(createTableSQL)
	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) UpsertExtension(ext *ExtensionDB) error {
	query := `
		INSERT OR REPLACE INTO extensions (
			id, name, display_name, description, version, publisher, engines, categories, tags,
			icon, repository, homepage, bugs, license, file_size, last_updated, file_path,
			verified, average_rating, review_count, download_count, namespace, extension_id,
			short_description, published_date, release_date, pre_release, deprecated,
			target_platform, readme_content, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		ext.ID, ext.Name, ext.DisplayName, ext.Description, ext.Version, ext.Publisher,
		ext.Engines, ext.Categories, ext.Tags, ext.Icon, ext.Repository, ext.Homepage,
		ext.Bugs, ext.License, ext.FileSize, ext.LastUpdated, ext.FilePath, ext.Verified,
		ext.AverageRating, ext.ReviewCount, ext.DownloadCount, ext.Namespace, ext.ExtensionID,
		ext.ShortDescription, ext.PublishedDate, ext.ReleaseDate, ext.PreRelease, ext.Deprecated,
		ext.TargetPlatform, ext.ReadmeContent, ext.CreatedAt, ext.UpdatedAt,
	)

	return err
}

func (d *Database) GetExtensionByID(id string) (*ExtensionDB, error) {
	query := `SELECT * FROM extensions WHERE id = ?`

	var ext ExtensionDB
	err := d.db.QueryRow(query, id).Scan(
		&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Version, &ext.Publisher,
		&ext.Engines, &ext.Categories, &ext.Tags, &ext.Icon, &ext.Repository, &ext.Homepage,
		&ext.Bugs, &ext.License, &ext.FileSize, &ext.LastUpdated, &ext.FilePath, &ext.CreatedAt,
		&ext.UpdatedAt, &ext.Verified, &ext.AverageRating, &ext.ReviewCount, &ext.DownloadCount,
		&ext.Namespace, &ext.ExtensionID, &ext.ShortDescription, &ext.PublishedDate, &ext.ReleaseDate,
		&ext.PreRelease, &ext.Deprecated, &ext.TargetPlatform, &ext.ReadmeContent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &ext, nil
}

func (d *Database) GetAllExtensions(page, limit int) ([]ExtensionDB, int64, error) {
	// Get total count
	var total int64
	err := d.db.QueryRow("SELECT COUNT(*) FROM extensions").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get extensions with pagination
	offset := (page - 1) * limit
	query := `SELECT * FROM extensions ORDER BY last_updated DESC LIMIT ? OFFSET ?`

	rows, err := d.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var extensions []ExtensionDB
	for rows.Next() {
		var ext ExtensionDB
		err := rows.Scan(
			&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Version, &ext.Publisher,
			&ext.Engines, &ext.Categories, &ext.Tags, &ext.Icon, &ext.Repository, &ext.Homepage,
			&ext.Bugs, &ext.License, &ext.FileSize, &ext.LastUpdated, &ext.FilePath, &ext.CreatedAt,
			&ext.UpdatedAt, &ext.Verified, &ext.AverageRating, &ext.ReviewCount, &ext.DownloadCount,
			&ext.Namespace, &ext.ExtensionID, &ext.ShortDescription, &ext.PublishedDate, &ext.ReleaseDate,
			&ext.PreRelease, &ext.Deprecated, &ext.TargetPlatform, &ext.ReadmeContent,
		)
		if err != nil {
			return nil, 0, err
		}
		extensions = append(extensions, ext)
	}

	return extensions, total, nil
}

func (d *Database) SearchExtensions(query string, page, limit int) ([]ExtensionDB, int64, error) {
	searchPattern := "%" + query + "%"

	// Get total count
	countQuery := `SELECT COUNT(*) FROM extensions 
		WHERE name LIKE ? OR display_name LIKE ? OR description LIKE ? OR publisher LIKE ?`

	var total int64
	err := d.db.QueryRow(countQuery, searchPattern, searchPattern, searchPattern, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get extensions with search and pagination
	offset := (page - 1) * limit
	searchQuery := `SELECT * FROM extensions 
		WHERE name LIKE ? OR display_name LIKE ? OR description LIKE ? OR publisher LIKE ?
		ORDER BY last_updated DESC LIMIT ? OFFSET ?`

	rows, err := d.db.Query(searchQuery, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var extensions []ExtensionDB
	for rows.Next() {
		var ext ExtensionDB
		err := rows.Scan(
			&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Version, &ext.Publisher,
			&ext.Engines, &ext.Categories, &ext.Tags, &ext.Icon, &ext.Repository, &ext.Homepage,
			&ext.Bugs, &ext.License, &ext.FileSize, &ext.LastUpdated, &ext.FilePath, &ext.CreatedAt,
			&ext.UpdatedAt, &ext.Verified, &ext.AverageRating, &ext.ReviewCount, &ext.DownloadCount,
			&ext.Namespace, &ext.ExtensionID, &ext.ShortDescription, &ext.PublishedDate, &ext.ReleaseDate,
			&ext.PreRelease, &ext.Deprecated, &ext.TargetPlatform, &ext.ReadmeContent,
		)
		if err != nil {
			return nil, 0, err
		}
		extensions = append(extensions, ext)
	}

	return extensions, total, nil
}

func (d *Database) DeleteExtension(id string) error {
	query := `DELETE FROM extensions WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) DeleteAllExtensions() error {
	query := `DELETE FROM extensions`
	_, err := d.db.Exec(query)
	return err
}

func (d *Database) GetStats() (map[string]interface{}, error) {
	var total int64
	err := d.db.QueryRow("SELECT COUNT(*) FROM extensions").Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get publishers count
	publishersQuery := `SELECT publisher, COUNT(*) as count FROM extensions GROUP BY publisher`
	rows, err := d.db.Query(publishersQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	publishersMap := make(map[string]int64)
	for rows.Next() {
		var publisher string
		var count int64
		if err := rows.Scan(&publisher, &count); err != nil {
			return nil, err
		}
		publishersMap[publisher] = count
	}

	// Get categories count (simplified - counting non-empty categories)
	categoriesQuery := `SELECT COUNT(*) FROM extensions WHERE categories IS NOT NULL AND categories != ''`
	var categoriesCount int64
	err = d.db.QueryRow(categoriesQuery).Scan(&categoriesCount)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_extensions": total,
		"publishers":       publishersMap,
		"categories":       map[string]int64{"total": categoriesCount},
	}, nil
}

func (d *Database) GetExtensionByFilePath(filePath string) (*ExtensionDB, error) {
	query := `SELECT * FROM extensions WHERE file_path = ?`

	var ext ExtensionDB
	err := d.db.QueryRow(query, filePath).Scan(
		&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Version, &ext.Publisher,
		&ext.Engines, &ext.Categories, &ext.Tags, &ext.Icon, &ext.Repository, &ext.Homepage,
		&ext.Bugs, &ext.License, &ext.FileSize, &ext.LastUpdated, &ext.FilePath, &ext.CreatedAt,
		&ext.UpdatedAt, &ext.Verified, &ext.AverageRating, &ext.ReviewCount, &ext.DownloadCount,
		&ext.Namespace, &ext.ExtensionID, &ext.ShortDescription, &ext.PublishedDate, &ext.ReleaseDate,
		&ext.PreRelease, &ext.Deprecated, &ext.TargetPlatform, &ext.ReadmeContent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &ext, nil
}

func (d *Database) GetExtensionsByPublisher(publisher string, page, limit int) ([]ExtensionDB, int64, error) {
	// Get total count
	var total int64
	err := d.db.QueryRow("SELECT COUNT(*) FROM extensions WHERE publisher = ?", publisher).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get extensions with pagination
	offset := (page - 1) * limit
	query := `SELECT * FROM extensions WHERE publisher = ? ORDER BY last_updated DESC LIMIT ? OFFSET ?`

	rows, err := d.db.Query(query, publisher, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var extensions []ExtensionDB
	for rows.Next() {
		var ext ExtensionDB
		err := rows.Scan(
			&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Version, &ext.Publisher,
			&ext.Engines, &ext.Categories, &ext.Tags, &ext.Icon, &ext.Repository, &ext.Homepage,
			&ext.Bugs, &ext.License, &ext.FileSize, &ext.LastUpdated, &ext.FilePath, &ext.CreatedAt,
			&ext.UpdatedAt, &ext.Verified, &ext.AverageRating, &ext.ReviewCount, &ext.DownloadCount,
			&ext.Namespace, &ext.ExtensionID, &ext.ShortDescription, &ext.PublishedDate, &ext.ReleaseDate,
			&ext.PreRelease, &ext.Deprecated, &ext.TargetPlatform, &ext.ReadmeContent,
		)
		if err != nil {
			return nil, 0, err
		}
		extensions = append(extensions, ext)
	}

	return extensions, total, nil
}

func (d *Database) GetDB() *sql.DB {
	return d.db
}
