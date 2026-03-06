package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
)

func TestConfigShow(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	config.SetDefaults()

	var buf bytes.Buffer
	configCmd.SetOut(&buf)

	if err := runConfigShow(configCmd, nil); err != nil {
		t.Fatalf("config command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Config file:") {
		t.Error("output missing 'Config file:' header")
	}

	keys := []string{
		"refresh:", "log_dir:", "log_max_size:", "log_max_files:",
		"auto_enabled:", "auto_snapshot_interval:", "thin_age_threshold:", "thin_cadence:",
	}
	for _, key := range keys {
		if !strings.Contains(output, key) {
			t.Errorf("output missing key %q", key)
		}
	}
}

func TestConfigInitCreatesFile(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	target := filepath.Join(t.TempDir(), "config.yaml")
	oldCfgFile := cfgFile
	cfgFile = target
	defer func() { cfgFile = oldCfgFile }()

	var buf bytes.Buffer
	configInitCmd.SetOut(&buf)

	if err := runConfigInit(configInitCmd, nil); err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	if !strings.Contains(buf.String(), "Config file created:") {
		t.Error("output missing success message")
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading created file: %v", err)
	}
	if !strings.Contains(string(content), "#") {
		t.Error("created file missing comment lines")
	}
	if !strings.Contains(string(content), "refresh:") {
		t.Error("created file missing expected YAML content")
	}
}

func TestConfigInitCreatesDirectory(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	target := filepath.Join(t.TempDir(), "new", "subdir", "config.yaml")
	oldCfgFile := cfgFile
	cfgFile = target
	defer func() { cfgFile = oldCfgFile }()

	var buf bytes.Buffer
	configInitCmd.SetOut(&buf)

	if err := runConfigInit(configInitCmd, nil); err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Errorf("expected file at %s, got error: %v", target, err)
	}
}

func TestConfigInitErrorsOnExistingFile(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	target := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("creating pre-existing file: %v", err)
	}

	oldCfgFile := cfgFile
	cfgFile = target
	defer func() { cfgFile = oldCfgFile }()

	err := runConfigInit(configInitCmd, nil)
	if err == nil {
		t.Fatal("expected error for existing file, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q does not mention 'already exists'", err.Error())
	}
}

func TestConfigInitUsesDefaultPath(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	oldCfgFile := cfgFile
	cfgFile = ""
	defer func() { cfgFile = oldCfgFile }()

	var buf bytes.Buffer
	configInitCmd.SetOut(&buf)

	if err := runConfigInit(configInitCmd, nil); err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	expected := filepath.Join(tmpHome, ".config", "snappy", "config.yaml")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected default config file at %s, got error: %v", expected, err)
	}
}
