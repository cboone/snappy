package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

// mockRunner returns canned output based on the command name and args.
type mockRunner struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	output []byte
	err    error
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	return nil, fmt.Errorf("unmocked command: %s", key)
}

// setFlag sets a flag value on a command, failing the test if it errors.
//
//nolint:unparam // name is always "json" today, but this helper is general-purpose.
func setFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("setting flag --%s=%s: %v", name, value, err)
	}
}
