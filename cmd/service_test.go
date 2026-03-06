package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/service"
)

func TestServiceCommandStructure(t *testing.T) {
	if serviceCmd.Use != "service" {
		t.Errorf("serviceCmd.Use = %q, want %q", serviceCmd.Use, "service")
	}

	subcommands := map[string]bool{
		"install":   false,
		"uninstall": false,
		"start":     false,
		"stop":      false,
		"status":    false,
		"log":       false,
	}

	for _, sub := range serviceCmd.Commands() {
		if _, ok := subcommands[sub.Name()]; ok {
			subcommands[sub.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("service subcommand %q not registered", name)
		}
	}
}

func TestServiceCommandRegistered(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "service" {
			found = true
			break
		}
	}
	if !found {
		t.Error("service command not registered on root")
	}
}

func TestFindTail(t *testing.T) {
	path, err := findTail()
	if err != nil {
		t.Fatalf("findTail() error = %v", err)
	}
	if path == "" {
		t.Error("findTail() returned empty path")
	}
}

func TestRunServiceStatusWarnsOnBinaryMismatch(t *testing.T) {
	binDir := t.TempDir()
	installedPath := filepath.Join(binDir, "snappy-installed")
	currentPath := filepath.Join(binDir, "snappy-current")

	if err := os.WriteFile(installedPath, []byte("installed"), 0o755); err != nil {
		t.Fatalf("writing installed binary: %v", err)
	}
	if err := os.WriteFile(currentPath, []byte("current"), 0o755); err != nil {
		t.Fatalf("writing current binary: %v", err)
	}

	origServiceStatus := serviceStatus
	origResolveBinaryPath := resolveServiceBinaryPath
	t.Cleanup(func() {
		serviceStatus = origServiceStatus
		resolveServiceBinaryPath = origResolveBinaryPath
	})

	serviceStatus = func(string) (*service.Info, error) {
		return &service.Info{
			Installed:  true,
			Running:    true,
			PID:        12345,
			BinaryPath: installedPath,
			PlistPath:  "/tmp/com.cboone.snappy.plist",
			Label:      service.DefaultLabel,
		}, nil
	}
	resolveServiceBinaryPath = func() (string, error) {
		return currentPath, nil
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := runServiceStatus(cmd, nil); err != nil {
		t.Fatalf("runServiceStatus() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "warning: installed service uses a different binary") {
		t.Fatalf("output missing mismatch warning:\n%s", got)
	}
	if !strings.Contains(got, currentPath) {
		t.Fatalf("output missing current binary path:\n%s", got)
	}
	if !strings.Contains(got, "snappy service install") {
		t.Fatalf("output missing reinstall guidance:\n%s", got)
	}
}

func TestRunServiceStatusNoMismatchWarningWhenPathsMatch(t *testing.T) {
	binDir := t.TempDir()
	binaryPath := filepath.Join(binDir, "snappy")
	if err := os.WriteFile(binaryPath, []byte("snappy"), 0o755); err != nil {
		t.Fatalf("writing binary: %v", err)
	}

	origServiceStatus := serviceStatus
	origResolveBinaryPath := resolveServiceBinaryPath
	t.Cleanup(func() {
		serviceStatus = origServiceStatus
		resolveServiceBinaryPath = origResolveBinaryPath
	})

	serviceStatus = func(string) (*service.Info, error) {
		return &service.Info{
			Installed:  true,
			Running:    true,
			PID:        12345,
			BinaryPath: binaryPath,
			PlistPath:  "/tmp/com.cboone.snappy.plist",
			Label:      service.DefaultLabel,
		}, nil
	}
	resolveServiceBinaryPath = func() (string, error) {
		return binaryPath, nil
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := runServiceStatus(cmd, nil); err != nil {
		t.Fatalf("runServiceStatus() error = %v", err)
	}

	got := out.String()
	if strings.Contains(got, "warning: installed service uses a different binary") {
		t.Fatalf("output unexpectedly contains mismatch warning:\n%s", got)
	}
}
