// Package config handles configuration loading for Snappy.
package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime settings for Snappy.
type Config struct {
	RefreshInterval      time.Duration
	MountPoint           string
	LogDir               string
	AutoEnabled          bool
	AutoSnapshotInterval time.Duration
	ThinAgeThreshold     time.Duration
	ThinCadence          time.Duration
}

// Load reads configuration from Viper, applying defaults for any
// values not set via environment variables or config file.
func Load() *Config {
	return &Config{
		RefreshInterval:      viper.GetDuration("refresh"),
		MountPoint:           viper.GetString("mount"),
		LogDir:               viper.GetString("log_dir"),
		AutoEnabled:          viper.GetBool("auto_enabled"),
		AutoSnapshotInterval: 60 * time.Second,
		ThinAgeThreshold:     600 * time.Second,
		ThinCadence:          300 * time.Second,
	}
}

// SetDefaults registers default values with Viper. Call this during
// Cobra's initConfig phase before Load.
func SetDefaults() {
	viper.SetDefault("refresh", 60*time.Second)
	viper.SetDefault("mount", "/")
	viper.SetDefault("log_dir", "")
	viper.SetDefault("auto_enabled", true)
}
