package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

type Config struct {
	GeniusAPIKey     string `mapstructure:"genius_api_key"`
	MusixmatchAPIKey string `mapstructure:"musixmatch_api_key"`
	LastFMAPIKey     string `mapstructure:"lastfm_api_key"`
	DiscogsToken     string `mapstructure:"discogs_token"`

	DefaultDirectory string `mapstructure:"default_directory"`
	MaxFileSize      int64  `mapstructure:"max_file_size"`
	EnableLogging    bool   `mapstructure:"enable_logging"`
	LogLevel         string `mapstructure:"log_level"`

	ShowFileSize    bool   `mapstructure:"show_file_size"`

	PreferredLyricsSource  string `mapstructure:"preferred_lyrics_source"`
	PreferredArtworkSource string `mapstructure:"preferred_artwork_source"`
	MaxArtworkSize         int    `mapstructure:"max_artwork_size"`
	EnableImageResize      bool   `mapstructure:"enable_image_resize"`

	BatchConcurrency int `mapstructure:"batch_concurrency"`
	BatchTimeout     int `mapstructure:"batch_timeout"`
}

func DefaultConfig() *Config {
	return &Config{
		DefaultDirectory:        ".",
		MaxFileSize:            100 * 1024 * 1024, // 100MB
		EnableLogging:          true,
		LogLevel:               "info",
		ShowFileSize:           true,
		PreferredLyricsSource:  "lyrics.ovh",
		PreferredArtworkSource: "itunes",
		MaxArtworkSize:         500,
		EnableImageResize:      true,
		BatchConcurrency:       5,
		BatchTimeout:           30,
	}
}

func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("failed to find home directory: %w", err)
	}

	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName(".tagTonic")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return config, nil
}

func SaveConfig(config *Config) error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to find home directory: %w", err)
	}

	configDir := filepath.Join(home, ".tagTonic")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	viper.SetConfigFile(configFile)

	viper.Set("genius_api_key", config.GeniusAPIKey)
	viper.Set("musixmatch_api_key", config.MusixmatchAPIKey)
	viper.Set("lastfm_api_key", config.LastFMAPIKey)
	viper.Set("discogs_token", config.DiscogsToken)
	viper.Set("default_directory", config.DefaultDirectory)
	viper.Set("max_file_size", config.MaxFileSize)
	viper.Set("enable_logging", config.EnableLogging)
	viper.Set("log_level", config.LogLevel)
	viper.Set("show_file_size", config.ShowFileSize)
	viper.Set("preferred_lyrics_source", config.PreferredLyricsSource)
	viper.Set("preferred_artwork_source", config.PreferredArtworkSource)
	viper.Set("max_artwork_size", config.MaxArtworkSize)
	viper.Set("enable_image_resize", config.EnableImageResize)
	viper.Set("batch_concurrency", config.BatchConcurrency)
	viper.Set("batch_timeout", config.BatchTimeout)

	return viper.WriteConfig()
}

func GetConfigPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tagTonic", "config.yaml"), nil
}

func CreateDefaultConfig() error {
	config := DefaultConfig()
	return SaveConfig(config)
}

func ValidateConfig(config *Config) error {
	validLogLevels := []string{"debug", "info", "warn", "error"}
	found := false
	for _, level := range validLogLevels {
		if config.LogLevel == level {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid log level: %s", config.LogLevel)
	}

	validLyricsSources := []string{"lyrics.ovh", "genius", "musixmatch"}
	found = false
	for _, source := range validLyricsSources {
		if config.PreferredLyricsSource == source {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid preferred lyrics source: %s", config.PreferredLyricsSource)
	}

	validArtworkSources := []string{"itunes", "discogs", "lastfm", "coverartarchive"}
	found = false
	for _, source := range validArtworkSources {
		if config.PreferredArtworkSource == source {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid preferred artwork source: %s", config.PreferredArtworkSource)
	}

	return nil
}
