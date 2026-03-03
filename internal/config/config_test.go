package config

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestLoadRefreshIntervalParsing(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  time.Duration
	}{
		{
			name:  "duration string",
			value: "2m",
			want:  2 * time.Minute,
		},
		{
			name:  "numeric string treated as seconds",
			value: "60",
			want:  60 * time.Second,
		},
		{
			name:  "integer treated as seconds",
			value: 90,
			want:  90 * time.Second,
		},
		{
			name:  "invalid value falls back to default",
			value: "not-a-duration",
			want:  60 * time.Second,
		},
		{
			name:  "zero falls back to default",
			value: "0",
			want:  60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			SetDefaults()
			viper.Set("refresh", tt.value)

			cfg := Load()
			if cfg.RefreshInterval != tt.want {
				t.Errorf("RefreshInterval = %v, want %v", cfg.RefreshInterval, tt.want)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	viper.Reset()
	SetDefaults()

	cfg := Load()

	if cfg.RefreshInterval != 60*time.Second {
		t.Errorf("RefreshInterval = %v, want %v", cfg.RefreshInterval, 60*time.Second)
	}
	if cfg.MountPoint != "/" {
		t.Errorf("MountPoint = %q, want %q", cfg.MountPoint, "/")
	}
	if cfg.LogDir != "" {
		t.Errorf("LogDir = %q, want %q", cfg.LogDir, "")
	}
	if cfg.LogMaxSize != 5*1024*1024 {
		t.Errorf("LogMaxSize = %d, want %d", cfg.LogMaxSize, 5*1024*1024)
	}
	if cfg.LogMaxFiles != 3 {
		t.Errorf("LogMaxFiles = %d, want %d", cfg.LogMaxFiles, 3)
	}
	if cfg.AutoEnabled != true {
		t.Errorf("AutoEnabled = %v, want %v", cfg.AutoEnabled, true)
	}
	if cfg.AutoSnapshotInterval != 60*time.Second {
		t.Errorf("AutoSnapshotInterval = %v, want %v", cfg.AutoSnapshotInterval, 60*time.Second)
	}
	if cfg.ThinAgeThreshold != 600*time.Second {
		t.Errorf("ThinAgeThreshold = %v, want %v", cfg.ThinAgeThreshold, 600*time.Second)
	}
	if cfg.ThinCadence != 300*time.Second {
		t.Errorf("ThinCadence = %v, want %v", cfg.ThinCadence, 300*time.Second)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		envVal  string
		checkFn func(*Config) bool
		desc    string
	}{
		{
			name:    "mount override",
			envVar:  "SNAPPY_MOUNT",
			envVal:  "/Volumes/Backup",
			checkFn: func(c *Config) bool { return c.MountPoint == "/Volumes/Backup" },
			desc:    "MountPoint",
		},
		{
			name:    "log_dir override",
			envVar:  "SNAPPY_LOG_DIR",
			envVal:  "/tmp/logs",
			checkFn: func(c *Config) bool { return c.LogDir == "/tmp/logs" },
			desc:    "LogDir",
		},
		{
			name:    "log_max_size override",
			envVar:  "SNAPPY_LOG_MAX_SIZE",
			envVal:  "1048576",
			checkFn: func(c *Config) bool { return c.LogMaxSize == 1048576 },
			desc:    "LogMaxSize",
		},
		{
			name:    "log_max_files override",
			envVar:  "SNAPPY_LOG_MAX_FILES",
			envVal:  "5",
			checkFn: func(c *Config) bool { return c.LogMaxFiles == 5 },
			desc:    "LogMaxFiles",
		},
		{
			name:    "auto_enabled override",
			envVar:  "SNAPPY_AUTO_ENABLED",
			envVal:  "false",
			checkFn: func(c *Config) bool { return c.AutoEnabled == false },
			desc:    "AutoEnabled",
		},
		{
			name:    "refresh override",
			envVar:  "SNAPPY_REFRESH",
			envVal:  "120",
			checkFn: func(c *Config) bool { return c.RefreshInterval == 120*time.Second },
			desc:    "RefreshInterval",
		},
		{
			name:    "auto_snapshot_interval override",
			envVar:  "SNAPPY_AUTO_SNAPSHOT_INTERVAL",
			envVal:  "30",
			checkFn: func(c *Config) bool { return c.AutoSnapshotInterval == 30*time.Second },
			desc:    "AutoSnapshotInterval",
		},
		{
			name:    "thin_age_threshold override",
			envVar:  "SNAPPY_THIN_AGE_THRESHOLD",
			envVal:  "900",
			checkFn: func(c *Config) bool { return c.ThinAgeThreshold == 900*time.Second },
			desc:    "ThinAgeThreshold",
		},
		{
			name:    "thin_cadence override",
			envVar:  "SNAPPY_THIN_CADENCE",
			envVal:  "600",
			checkFn: func(c *Config) bool { return c.ThinCadence == 600*time.Second },
			desc:    "ThinCadence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			SetDefaults()
			viper.SetEnvPrefix("SNAPPY")
			viper.AutomaticEnv()
			t.Setenv(tt.envVar, tt.envVal)

			cfg := Load()
			if !tt.checkFn(cfg) {
				t.Errorf("%s not overridden by %s=%s", tt.desc, tt.envVar, tt.envVal)
			}
		})
	}
}

func TestLoadDurationFieldParsing(t *testing.T) {
	type durationField struct {
		name     string
		viperKey string
		getFn    func(*Config) time.Duration
		fallback time.Duration
	}

	fields := []durationField{
		{"AutoSnapshotInterval", "auto_snapshot_interval", func(c *Config) time.Duration { return c.AutoSnapshotInterval }, 60 * time.Second},
		{"ThinAgeThreshold", "thin_age_threshold", func(c *Config) time.Duration { return c.ThinAgeThreshold }, 600 * time.Second},
		{"ThinCadence", "thin_cadence", func(c *Config) time.Duration { return c.ThinCadence }, 300 * time.Second},
	}

	cases := []struct {
		name  string
		value any
		want  func(fallback time.Duration) time.Duration
	}{
		{
			name:  "duration string",
			value: "2m",
			want:  func(_ time.Duration) time.Duration { return 2 * time.Minute },
		},
		{
			name:  "numeric string treated as seconds",
			value: "45",
			want:  func(_ time.Duration) time.Duration { return 45 * time.Second },
		},
		{
			name:  "integer treated as seconds",
			value: 90,
			want:  func(_ time.Duration) time.Duration { return 90 * time.Second },
		},
		{
			name:  "invalid value falls back to default",
			value: "not-a-duration",
			want:  func(fb time.Duration) time.Duration { return fb },
		},
		{
			name:  "zero falls back to default",
			value: "0",
			want:  func(fb time.Duration) time.Duration { return fb },
		},
		{
			name:  "negative falls back to default",
			value: "-10",
			want:  func(fb time.Duration) time.Duration { return fb },
		},
	}

	for _, f := range fields {
		for _, tc := range cases {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				viper.Reset()
				SetDefaults()
				viper.Set(f.viperKey, tc.value)

				cfg := Load()
				got := f.getFn(cfg)
				want := tc.want(f.fallback)
				if got != want {
					t.Errorf("%s = %v, want %v", f.name, got, want)
				}
			})
		}
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error: %v", err)
	}
	if !strings.HasPrefix(path, home) {
		t.Errorf("path %q does not start with home directory %q", path, home)
	}
	if !strings.HasSuffix(path, ".config/snappy/config.yaml") {
		t.Errorf("path %q does not end with .config/snappy/config.yaml", path)
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDefaultConfig(&buf); err != nil {
		t.Fatalf("WriteDefaultConfig() error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("WriteDefaultConfig() produced empty output")
	}

	if !strings.Contains(output, "#") {
		t.Error("output does not contain comment lines")
	}

	keys := []string{
		"refresh:", "mount:", "log_dir:", "log_max_size:", "log_max_files:",
		"auto_enabled:", "auto_snapshot_interval:", "thin_age_threshold:", "thin_cadence:",
	}
	for _, key := range keys {
		if !strings.Contains(output, key) {
			t.Errorf("output missing key %q", key)
		}
	}
}

func TestFormatConfig(t *testing.T) {
	cfg := &Config{
		RefreshInterval:      60 * time.Second,
		MountPoint:           "/",
		LogDir:               "",
		LogMaxSize:           5 * 1024 * 1024,
		LogMaxFiles:          3,
		AutoEnabled:          true,
		AutoSnapshotInterval: 60 * time.Second,
		ThinAgeThreshold:     600 * time.Second,
		ThinCadence:          300 * time.Second,
	}

	var buf bytes.Buffer
	if err := FormatConfig(&buf, cfg, "/some/config.yaml"); err != nil {
		t.Fatalf("FormatConfig() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Config file: /some/config.yaml") {
		t.Error("output missing config file path")
	}

	expected := []string{
		"refresh: 1m0s",
		"mount: /",
		"log_max_size: 5242880",
		"log_max_files: 3",
		"auto_enabled: true",
		"auto_snapshot_interval: 1m0s",
		"thin_age_threshold: 10m0s",
		"thin_cadence: 5m0s",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestFormatConfigNoFile(t *testing.T) {
	cfg := &Config{
		RefreshInterval:      60 * time.Second,
		AutoSnapshotInterval: 60 * time.Second,
		ThinAgeThreshold:     600 * time.Second,
		ThinCadence:          300 * time.Second,
	}

	var buf bytes.Buffer
	if err := FormatConfig(&buf, cfg, ""); err != nil {
		t.Fatalf("FormatConfig() error: %v", err)
	}

	if !strings.Contains(buf.String(), "Config file: none") {
		t.Error("output should show 'none' when no config file is used")
	}
}

func TestLoadWithoutSetDefaults(t *testing.T) {
	viper.Reset()

	cfg := Load()

	// Duration fields use Load()'s baked-in fallbacks even without SetDefaults.
	if cfg.RefreshInterval != 60*time.Second {
		t.Errorf("RefreshInterval = %v, want %v", cfg.RefreshInterval, 60*time.Second)
	}
	if cfg.AutoSnapshotInterval != 60*time.Second {
		t.Errorf("AutoSnapshotInterval = %v, want %v", cfg.AutoSnapshotInterval, 60*time.Second)
	}
	if cfg.ThinAgeThreshold != 600*time.Second {
		t.Errorf("ThinAgeThreshold = %v, want %v", cfg.ThinAgeThreshold, 600*time.Second)
	}
	if cfg.ThinCadence != 300*time.Second {
		t.Errorf("ThinCadence = %v, want %v", cfg.ThinCadence, 300*time.Second)
	}

	// Numeric and string fields get zero values without SetDefaults.
	if cfg.LogMaxSize != 0 {
		t.Errorf("LogMaxSize = %d, want 0", cfg.LogMaxSize)
	}
	if cfg.LogMaxFiles != 0 {
		t.Errorf("LogMaxFiles = %d, want 0", cfg.LogMaxFiles)
	}
	if cfg.MountPoint != "" {
		t.Errorf("MountPoint = %q, want %q", cfg.MountPoint, "")
	}
	if cfg.LogDir != "" {
		t.Errorf("LogDir = %q, want %q", cfg.LogDir, "")
	}
	if cfg.AutoEnabled != false {
		t.Errorf("AutoEnabled = %v, want %v", cfg.AutoEnabled, false)
	}
}
