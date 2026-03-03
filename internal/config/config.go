// Package config handles configuration loading for Snappy.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime settings for Snappy.
type Config struct {
	RefreshInterval      time.Duration
	MountPoint           string
	LogDir               string
	LogMaxSize           int64 // max log file size in bytes; 0 = no rotation
	LogMaxFiles          int   // number of rotated backup files to keep
	AutoEnabled          bool
	AutoSnapshotInterval time.Duration
	ThinAgeThreshold     time.Duration
	ThinCadence          time.Duration
}

const (
	defaultMount       = "/"
	defaultLogDir      = ""
	defaultLogMaxSize  = int64(5 * 1024 * 1024) // 5 MB
	defaultLogMaxFiles = 3
	defaultAutoEnabled = true
)

const (
	defaultRefreshInterval      = 60 * time.Second
	defaultAutoSnapshotInterval = 60 * time.Second
	defaultThinAgeThreshold     = 600 * time.Second
	defaultThinCadence          = 300 * time.Second
)

// Load reads configuration from Viper, applying defaults for any
// values not set via environment variables or config file.
func Load() *Config {
	cfg := &Config{
		RefreshInterval:      parseSecondsOrDuration(viper.Get("refresh"), defaultRefreshInterval),
		MountPoint:           viper.GetString("mount"),
		LogDir:               viper.GetString("log_dir"),
		LogMaxSize:           viper.GetInt64("log_max_size"),
		LogMaxFiles:          viper.GetInt("log_max_files"),
		AutoEnabled:          viper.GetBool("auto_enabled"),
		AutoSnapshotInterval: parseSecondsOrDuration(viper.Get("auto_snapshot_interval"), defaultAutoSnapshotInterval),
		ThinAgeThreshold:     parseSecondsOrDuration(viper.Get("thin_age_threshold"), defaultThinAgeThreshold),
		ThinCadence:          parseSecondsOrDuration(viper.Get("thin_cadence"), defaultThinCadence),
	}

	if cfg.LogDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.LogDir = filepath.Join(home, ".local", "share", "snappy")
		}
	}

	return cfg
}

// SetDefaults registers default values with Viper. Call this during
// Cobra's initConfig phase before Load.
func SetDefaults() {
	viper.SetDefault("refresh", defaultRefreshInterval)
	viper.SetDefault("mount", defaultMount)
	viper.SetDefault("log_dir", defaultLogDir)
	viper.SetDefault("log_max_size", defaultLogMaxSize)
	viper.SetDefault("log_max_files", defaultLogMaxFiles)
	viper.SetDefault("auto_enabled", defaultAutoEnabled)
	viper.SetDefault("auto_snapshot_interval", defaultAutoSnapshotInterval)
	viper.SetDefault("thin_age_threshold", defaultThinAgeThreshold)
	viper.SetDefault("thin_cadence", defaultThinCadence)
}

// DefaultConfigPath returns the default config file path:
// ~/.config/snappy/config.yaml.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".config", "snappy", "config.yaml"), nil
}

var defaultConfigTmpl = template.Must(template.New("config").Parse(`# Snappy configuration
# See: https://github.com/cboone/snappy

# How often to refresh the snapshot list.
# Accepts Go duration strings (e.g., "60s", "2m") or plain seconds.
refresh: {{.Refresh}}

# Mount point to monitor for Time Machine snapshots.
mount: "{{.Mount}}"

# Directory for log files. Leave empty for the default (~/.local/share/snappy).
log_dir: "{{.LogDir}}"

# Maximum log file size in bytes before rotation (default: 5 MB).
log_max_size: {{.LogMaxSize}}

# Number of rotated backup log files to keep.
log_max_files: {{.LogMaxFiles}}

# Whether to enable auto-snapshots at startup.
auto_enabled: {{.AutoEnabled}}

# Interval between automatic snapshots.
auto_snapshot_interval: {{.AutoSnapshotInterval}}

# Snapshots older than this threshold are candidates for thinning.
thin_age_threshold: {{.ThinAgeThreshold}}

# Minimum time gap to preserve between snapshots when thinning.
thin_cadence: {{.ThinCadence}}
`))

// WriteDefaultConfig writes a commented YAML template with default values to w.
func WriteDefaultConfig(w io.Writer) error {
	data := struct {
		Refresh              string
		Mount                string
		LogDir               string
		LogMaxSize           int64
		LogMaxFiles          int
		AutoEnabled          bool
		AutoSnapshotInterval string
		ThinAgeThreshold     string
		ThinCadence          string
	}{
		Refresh:              formatDurationAsSeconds(defaultRefreshInterval),
		Mount:                defaultMount,
		LogDir:               defaultLogDir,
		LogMaxSize:           defaultLogMaxSize,
		LogMaxFiles:          defaultLogMaxFiles,
		AutoEnabled:          defaultAutoEnabled,
		AutoSnapshotInterval: formatDurationAsSeconds(defaultAutoSnapshotInterval),
		ThinAgeThreshold:     formatDurationAsSeconds(defaultThinAgeThreshold),
		ThinCadence:          formatDurationAsSeconds(defaultThinCadence),
	}
	return defaultConfigTmpl.Execute(w, data)
}

func formatDurationAsSeconds(d time.Duration) string {
	if d%time.Second == 0 {
		return strconv.FormatInt(int64(d/time.Second), 10) + "s"
	}
	return d.String()
}

var formatConfigTmpl = template.Must(template.New("format").Parse(`Config file: {{.ConfigFile}}

refresh: {{.Refresh}}
mount: {{.Mount}}
log_dir:{{if .LogDir}} {{.LogDir}}{{end}}
log_max_size: {{.LogMaxSize}}
log_max_files: {{.LogMaxFiles}}
auto_enabled: {{.AutoEnabled}}
auto_snapshot_interval: {{.AutoSnapshotInterval}}
thin_age_threshold: {{.ThinAgeThreshold}}
thin_cadence: {{.ThinCadence}}
`))

// FormatConfig writes a human-readable display of the effective configuration to w.
func FormatConfig(w io.Writer, cfg *Config, configFile string) error {
	if configFile == "" {
		configFile = "none"
	}
	data := struct {
		ConfigFile           string
		Refresh              string
		Mount                string
		LogDir               string
		LogMaxSize           int64
		LogMaxFiles          int
		AutoEnabled          bool
		AutoSnapshotInterval string
		ThinAgeThreshold     string
		ThinCadence          string
	}{
		ConfigFile:           configFile,
		Refresh:              cfg.RefreshInterval.String(),
		Mount:                cfg.MountPoint,
		LogDir:               cfg.LogDir,
		LogMaxSize:           cfg.LogMaxSize,
		LogMaxFiles:          cfg.LogMaxFiles,
		AutoEnabled:          cfg.AutoEnabled,
		AutoSnapshotInterval: cfg.AutoSnapshotInterval.String(),
		ThinAgeThreshold:     cfg.ThinAgeThreshold.String(),
		ThinCadence:          cfg.ThinCadence.String(),
	}
	return formatConfigTmpl.Execute(w, data)
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
		return parseStringDuration(v, fallback)
	default:
		if f, ok := toFloat64(raw); ok {
			return secondsToDuration(f)
		}
		return fallback
	}
}

func parseStringDuration(s string, fallback time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}

	if seconds, err := strconv.ParseFloat(s, 64); err == nil {
		if seconds <= 0 {
			return fallback
		}
		return time.Duration(seconds * float64(time.Second))
	}

	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func toFloat64(raw any) (float64, bool) {
	rv := reflect.ValueOf(raw)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}
