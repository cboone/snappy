package config

import (
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

func TestLoadRefreshIntervalDefault(t *testing.T) {
	viper.Reset()
	SetDefaults()

	cfg := Load()
	if cfg.RefreshInterval != 60*time.Second {
		t.Errorf("RefreshInterval = %v, want 60s", cfg.RefreshInterval)
	}
}
