// Package platform wraps macOS system commands behind a testable interface.
package platform

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// OSRunner executes commands via os/exec.
type OSRunner struct{}

// Run executes the named command with the given arguments and returns its
// standard output.
func (r OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && stderr.Len() > 0 {
		return out, fmt.Errorf("%w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out, err
}
