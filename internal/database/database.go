package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"littlevsx/internal/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ExtensionDB struct {
	ID               string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Name             string    `gorm:"type:varchar(255);not null" json:"name"`
	DisplayName      string    `gorm:"type:varchar(500)" json:"displayName"`
	Description      string    `gorm:"type:text" json:"description"`
	Version          string    `gorm:"type:varchar(100);not null" json:"version"`
	Publisher        string    `gorm:"type:varchar(255);not null" json:"publisher"`
	Engines          string    `gorm:"type:text" json:"engines"`
	Categories       string    `gorm:"type:text" json:"categories"`
	Tags             string    `gorm:"type:text" json:"tags"`
	Icon             string    `gorm:"type:varchar(500)" json:"icon"`
	Repository       string    `gorm:"type:varchar(500)" json:"repository"`
	Homepage         string    `gorm:"type:varchar(500)" json:"homepage"`
	Bugs             string    `gorm:"type:varchar(500)" json:"bugs"`
	License          string    `gorm:"type:varchar(255)" json:"license"`
	FileSize         int64     `gorm:"not null" json:"fileSize"`
	LastUpdated      time.Time `gorm:"not null" json:"lastUpdated"`
	FilePath         string    `gorm:"type:varchar(1000);not null" json:"filePath"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	Verified         bool      `gorm:"default:true" json:"verified"`
	AverageRating    float64   `gorm:"default:5.0" json:"averageRating"`
	ReviewCount      int64     `gorm:"default:100" json:"reviewCount"`
	DownloadCount    int64     `gorm:"default:1000" json:"downloadCount"`
	Namespace        string    `gorm:"type:varchar(255)" json:"namespace"`
	ExtensionID      string    `gorm:"type:varchar(255)" json:"extensionId"`
	ShortDescription string    `gorm:"type:text" json:"shortDescription"`
	PublishedDate    time.Time `json:"publishedDate"`
	ReleaseDate      time.Time `json:"releaseDate"`
	PreRelease       bool      `gorm:"default:false" json:"preRelease"`
	Deprecated       bool      `gorm:"default:false" json:"deprecated"`
	TargetPlatform   string    `gorm:"type:varchar(50);default:'universal'" json:"targetPlatform"`
	ReadmeContent    string    `gorm:"type:text" json:"readmeContent"`
}

type Database struct {
	db *gorm.DB
}

func New() (*Database, error) {
	cfg := config.GetConfig()

	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	var gormLogger logger.Interface
	if cfg.LogQueries {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if cfg.AutoMigrate {
		if err := db.AutoMigrate(&ExtensionDB{}); err != nil {
			return nil, fmt.Errorf("database migration error: %w", err)
		}
		log.Println("Database migration completed")
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) UpsertExtension(ext *ExtensionDB) error {
	result := d.db.Save(ext)
	return result.Error
}

func (d *Database) GetExtensionByID(id string) (*ExtensionDB, error) {
	var ext ExtensionDB
	result := d.db.Where("id = ?", id).First(&ext)
	if result.Error != nil {
		return nil, result.Error
	}
	return &ext, nil
}

func (d *Database) GetAllExtensions(page, limit int) ([]ExtensionDB, int64, error) {
	var extensions []ExtensionDB
	var total int64

	if err := d.db.Model(&ExtensionDB{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	result := d.db.Offset(offset).Limit(limit).Find(&extensions)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return extensions, total, nil
}

func (d *Database) SearchExtensions(query string, page, limit int) ([]ExtensionDB, int64, error) {
	var extensions []ExtensionDB
	var total int64

	searchCondition := d.db.Where(
		"name LIKE ? OR display_name LIKE ? OR description LIKE ? OR publisher LIKE ?",
		"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%",
	)

	if err := searchCondition.Model(&ExtensionDB{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	result := searchCondition.Offset(offset).Limit(limit).Find(&extensions)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return extensions, total, nil
}

func (d *Database) DeleteExtension(id string) error {
	result := d.db.Where("id = ?", id).Delete(&ExtensionDB{})
	return result.Error
}

func (d *Database) DeleteAllExtensions() error {
	result := d.db.Where("1 = 1").Delete(&ExtensionDB{})
	return result.Error
}

func (d *Database) GetStats() (map[string]interface{}, error) {
	var total int64
	var publishers []struct {
		Publisher string `json:"publisher"`
		Count     int64  `json:"count"`
	}
	var categories []struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}

	if err := d.db.Model(&ExtensionDB{}).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := d.db.Model(&ExtensionDB{}).
		Select("publisher, count(*) as count").
		Group("publisher").
		Find(&publishers).Error; err != nil {
		return nil, err
	}

	if err := d.db.Model(&ExtensionDB{}).
		Select("categories, count(*) as count").
		Group("categories").
		Find(&categories).Error; err != nil {
		return nil, err
	}

	publishersMap := make(map[string]int64)
	for _, p := range publishers {
		publishersMap[p.Publisher] = p.Count
	}

	categoriesMap := make(map[string]int64)
	for _, c := range categories {
		if c.Category != "" {
			categoriesMap[c.Category] = c.Count
		}
	}

	return map[string]interface{}{
		"total_extensions": total,
		"publishers":       publishersMap,
		"categories":       categoriesMap,
	}, nil
}

func (d *Database) GetExtensionByFilePath(filePath string) (*ExtensionDB, error) {
	var ext ExtensionDB
	result := d.db.Where("file_path = ?", filePath).First(&ext)
	if result.Error != nil {
		return nil, result.Error
	}
	return &ext, nil
}

func (d *Database) GetExtensionsByPublisher(publisher string, page, limit int) ([]ExtensionDB, int64, error) {
	var extensions []ExtensionDB
	var total int64

	if err := d.db.Model(&ExtensionDB{}).Where("publisher = ?", publisher).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	result := d.db.Where("publisher = ?", publisher).Offset(offset).Limit(limit).Find(&extensions)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return extensions, total, nil
}

func (d *Database) GetDB() *gorm.DB {
	return d.db
}
