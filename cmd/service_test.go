package cmd

import (
	"testing"
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
