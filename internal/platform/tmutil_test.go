package platform

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestThinningSkipReason(t *testing.T) {
	tests := []struct {
		name   string
		output string
		err    error
		want   string
	}{
		{name: "idle", output: "Backup session status:\n{\n    Running = 0;\n}\n"},
		{name: "running", output: "Backup session status:\n{\n    Running = 1;\n}\n", want: "backup is running"},
		{name: "tabs", output: "\tRunning\t=\t1;\t\n", want: "backup is running"},
		{name: "missing", output: "Backup session status:\n{}", want: "invalid Running field"},
		{name: "empty", want: "invalid Running field"},
		{name: "unknown", output: "Running = 2;", want: "invalid Running field"},
		{name: "misleading key", output: "NotRunning = 0;", want: "invalid Running field"},
		{name: "duplicate", output: "Running = 0;\nRunning = 1;", want: "invalid Running field"},
		{name: "command failure", output: "Running = 0;", err: fmt.Errorf("status unavailable"), want: "status unavailable"},
		{name: "cancelled", err: context.Canceled, want: "context canceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockRunner{responses: map[string]mockResponse{
				"tmutil status": {output: []byte(tt.output), err: tt.err},
			}}
			got := ThinningSkipReason(context.Background(), r)
			if (tt.want == "" && got != "") || (tt.want != "" && !strings.Contains(got, tt.want)) {
				t.Errorf("ThinningSkipReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
