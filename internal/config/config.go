package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port     int
	Host     string
	UseHTTPS bool
	CertFile string
	KeyFile  string
	BaseURL  string

	DBPath      string
	AutoMigrate bool
	LogQueries  bool

	ExtensionsDir string

	AssetsDir       string
	AssetsCacheTime int
}

func GetConfig() Config {
	return Config{
		Port:     viper.GetInt("server.port"),
		Host:     viper.GetString("server.host"),
		UseHTTPS: viper.GetBool("server.https"),
		CertFile: viper.GetString("server.cert_file"),
		KeyFile:  viper.GetString("server.key_file"),
		BaseURL:  viper.GetString("server.base_url"),

		DBPath:      viper.GetString("database.path"),
		AutoMigrate: viper.GetBool("database.auto_migrate"),
		LogQueries:  viper.GetBool("database.log_queries"),

		ExtensionsDir: viper.GetString("extensions.directory"),

		AssetsDir:       viper.GetString("assets.directory"),
		AssetsCacheTime: viper.GetInt("assets.cache_time"),
	}
}
