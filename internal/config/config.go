// Package config handles configuration loading for Snappy.
package config

import (
	"strconv"
	"strings"
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
		RefreshInterval:      parseSecondsOrDuration(viper.Get("refresh"), 60*time.Second),
		MountPoint:           viper.GetString("mount"),
		LogDir:               viper.GetString("log_dir"),
		AutoEnabled:          viper.GetBool("auto_enabled"),
		AutoSnapshotInterval: parseSecondsOrDuration(viper.Get("auto_snapshot_interval"), 60*time.Second),
		ThinAgeThreshold:     parseSecondsOrDuration(viper.Get("thin_age_threshold"), 600*time.Second),
		ThinCadence:          parseSecondsOrDuration(viper.Get("thin_cadence"), 300*time.Second),
	}
}

// SetDefaults registers default values with Viper. Call this during
// Cobra's initConfig phase before Load.
func SetDefaults() {
	viper.SetDefault("refresh", 60*time.Second)
	viper.SetDefault("mount", "/")
	viper.SetDefault("log_dir", "")
	viper.SetDefault("auto_enabled", true)
	viper.SetDefault("auto_snapshot_interval", 60*time.Second)
	viper.SetDefault("thin_age_threshold", 600*time.Second)
	viper.SetDefault("thin_cadence", 300*time.Second)
}

func parseSecondsOrDuration(raw any, fallback time.Duration) time.Duration {
	secondsToDuration := func(v float64) time.Duration {
		if v <= 0 {
			return fallback
		}
		return time.Duration(v * float64(time.Second))
	}

	switch v := raw.(type) {
	case time.Duration:
		if v <= 0 {
			return fallback
		}
		return v
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return fallback
		}

		if seconds, err := strconv.ParseFloat(s, 64); err == nil {
			return secondsToDuration(seconds)
		}

		d, err := time.ParseDuration(s)
		if err != nil || d <= 0 {
			return fallback
		}
		return d
	case int:
		return secondsToDuration(float64(v))
	case int8:
		return secondsToDuration(float64(v))
	case int16:
		return secondsToDuration(float64(v))
	case int32:
		return secondsToDuration(float64(v))
	case int64:
		return secondsToDuration(float64(v))
	case uint:
		return secondsToDuration(float64(v))
	case uint8:
		return secondsToDuration(float64(v))
	case uint16:
		return secondsToDuration(float64(v))
	case uint32:
		return secondsToDuration(float64(v))
	case uint64:
		return secondsToDuration(float64(v))
	case float32:
		return secondsToDuration(float64(v))
	case float64:
		return secondsToDuration(v)
	default:
		return fallback
	}
}
