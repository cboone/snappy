//go:build darwin

// Package main is the entry point for the snappy CLI.
package main

import (
	"fmt"
	"os"

	"github.com/cboone/snappy/cmd"
)

var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
