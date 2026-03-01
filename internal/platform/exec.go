// Package platform wraps macOS system commands behind a testable interface.
package platform

import (
	"context"
	"os/exec"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// OSRunner executes commands via os/exec.
type OSRunner struct{}

// Run executes the named command with the given arguments and returns its
// combined stdout output.
func (r OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
