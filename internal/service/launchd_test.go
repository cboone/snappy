package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePlistBasic(t *testing.T) {
	cfg := PlistConfig{
		Label:      "com.cboone.snappy",
		BinaryPath: "/usr/local/bin/snappy",
		LogDir:     "/Users/test/.local/share/snappy",
	}

	data, err := GeneratePlist(cfg)
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	content := string(data)

	checks := []string{
		"<string>com.cboone.snappy</string>",
		"<string>/usr/local/bin/snappy</string>",
		"<string>run</string>",
		"<true/>",
		"<string>Background</string>",
		"/Users/test/.local/share/snappy/snappy-service.log",
		"<!DOCTYPE plist",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("plist missing %q", want)
		}
	}

	// Should NOT contain --config when ConfigFile is empty.
	if strings.Contains(content, "--config") {
		t.Error("plist contains --config when ConfigFile is empty")
	}
}

func TestGeneratePlistWithConfigFile(t *testing.T) {
	cfg := PlistConfig{
		Label:      "com.cboone.snappy",
		BinaryPath: "/usr/local/bin/snappy",
		LogDir:     "/Users/test/.local/share/snappy",
		ConfigFile: "/Users/test/.config/snappy/custom.yaml",
	}

	data, err := GeneratePlist(cfg)
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "--config") {
		t.Error("plist missing --config flag")
	}
	if !strings.Contains(content, "/Users/test/.config/snappy/custom.yaml") {
		t.Error("plist missing config file path")
	}
}

func TestGeneratePlistEscapesXMLChars(t *testing.T) {
	cfg := PlistConfig{
		Label:      "com.cboone.snappy",
		BinaryPath: "/usr/local/bin/snappy & co",
		LogDir:     "/Users/test/.local/share/snappy",
	}

	data, err := GeneratePlist(cfg)
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	content := string(data)
	// The ampersand should be XML-escaped.
	if !strings.Contains(content, "snappy &amp; co") {
		t.Error("plist did not XML-escape ampersand in BinaryPath")
	}
	if strings.Contains(content, "snappy & co</string>") {
		t.Error("plist contains unescaped ampersand")
	}
}

func TestPlistPath(t *testing.T) {
	path, err := PlistPath("com.cboone.snappy")
	if err != nil {
		t.Fatalf("PlistPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "Library", "LaunchAgents", "com.cboone.snappy.plist")
	if path != want {
		t.Errorf("PlistPath() = %q, want %q", path, want)
	}
}

func TestParsePIDFromPrint(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{
			name:   "running",
			output: "com.cboone.snappy = {\n\tpid = 12345\n\tstate = running\n}",
			want:   12345,
		},
		{
			name:   "not running",
			output: "com.cboone.snappy = {\n\tpid = -\n\tstate = not running\n}",
			want:   0,
		},
		{
			name:   "empty",
			output: "",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePIDFromPrint(tt.output)
			if got != tt.want {
				t.Errorf("parsePIDFromPrint() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseRuntimeFromPrint(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantRunning bool
		wantPID     int
	}{
		{
			name:        "running",
			output:      "com.cboone.snappy = {\n\tpid = 12345\n\tstate = running\n}",
			wantRunning: true,
			wantPID:     12345,
		},
		{
			name:        "loaded not running",
			output:      "com.cboone.snappy = {\n\tpid = -\n\tstate = not running\n}",
			wantRunning: false,
			wantPID:     0,
		},
		{
			name:        "missing pid",
			output:      "com.cboone.snappy = {\n\tstate = waiting\n}",
			wantRunning: false,
			wantPID:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			running, pid := parseRuntimeFromPrint(tt.output)
			if running != tt.wantRunning {
				t.Errorf("parseRuntimeFromPrint() running = %v, want %v", running, tt.wantRunning)
			}
			if pid != tt.wantPID {
				t.Errorf("parseRuntimeFromPrint() pid = %d, want %d", pid, tt.wantPID)
			}
		})
	}
}

func TestIsNotLoadedError(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"No such process", true},
		{"Could not find service \"com.cboone.snappy\" in domain for uid: 502", true},
		{"Operation not permitted", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isNotLoadedError(tt.output)
		if got != tt.want {
			t.Errorf("isNotLoadedError(%q) = %v, want %v", tt.output, got, tt.want)
		}
	}
}

func TestReadBinaryFromPlist(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cboone.snappy</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/homebrew/bin/snappy</string>
        <string>run</string>
    </array>
</dict>
</plist>`

	path := filepath.Join(t.TempDir(), "test.plist")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readBinaryFromPlist(path)
	want := "/opt/homebrew/bin/snappy"
	if got != want {
		t.Errorf("readBinaryFromPlist() = %q, want %q", got, want)
	}
}

func TestReadBinaryFromPlistMissing(t *testing.T) {
	got := readBinaryFromPlist("/nonexistent/path")
	if got != "" {
		t.Errorf("readBinaryFromPlist() = %q, want empty", got)
	}
}

func TestReadBinaryFromPlistMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.plist")
	if err := os.WriteFile(path, []byte("not a plist at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readBinaryFromPlist(path)
	if got != "" {
		t.Errorf("readBinaryFromPlist() = %q, want empty for malformed plist", got)
	}
}

func TestReadBinaryFromPlistEmptyArgs(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cboone.snappy</string>
    <key>ProgramArguments</key>
    <array>
    </array>
</dict>
</plist>`

	path := filepath.Join(t.TempDir(), "empty-args.plist")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readBinaryFromPlist(path)
	if got != "" {
		t.Errorf("readBinaryFromPlist() = %q, want empty for empty ProgramArguments", got)
	}
}

func TestReadBinaryFromPlistRoundTrip(t *testing.T) {
	cfg := PlistConfig{
		Label:      "com.cboone.snappy",
		BinaryPath: "/opt/homebrew/bin/snappy",
		LogDir:     "/Users/test/.local/share/snappy",
	}

	data, err := GeneratePlist(cfg)
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "roundtrip.plist")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := readBinaryFromPlist(path)
	if got != cfg.BinaryPath {
		t.Errorf("readBinaryFromPlist() = %q, want %q", got, cfg.BinaryPath)
	}
}

func TestReadLogPathFromPlistRoundTrip(t *testing.T) {
	cfg := PlistConfig{
		Label:      "com.cboone.snappy",
		BinaryPath: "/opt/homebrew/bin/snappy",
		LogDir:     "/Users/test/.local/share/snappy",
	}

	data, err := GeneratePlist(cfg)
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "roundtrip.plist")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := ReadLogPathFromPlist(path)
	want := "/Users/test/.local/share/snappy/snappy-service.log"
	if got != want {
		t.Errorf("ReadLogPathFromPlist() = %q, want %q", got, want)
	}
}

func TestReadLogPathFromPlistMissing(t *testing.T) {
	got := ReadLogPathFromPlist("/nonexistent/path.plist")
	if got != "" {
		t.Errorf("ReadLogPathFromPlist() = %q, want empty for nonexistent file", got)
	}
}

func TestLogPath(t *testing.T) {
	got := LogPath("/some/dir")
	want := "/some/dir/snappy-service.log"
	if got != want {
		t.Errorf("LogPath() = %q, want %q", got, want)
	}
}

func TestDomainTarget(t *testing.T) {
	dt := domainTarget()
	if !strings.HasPrefix(dt, "gui/") {
		t.Errorf("domainTarget() = %q, want prefix gui/", dt)
	}
}

func TestServiceTarget(t *testing.T) {
	st := serviceTarget("com.cboone.snappy")
	if !strings.HasPrefix(st, "gui/") {
		t.Errorf("serviceTarget() missing gui/ prefix: %q", st)
	}
	if !strings.HasSuffix(st, "/com.cboone.snappy") {
		t.Errorf("serviceTarget() missing label suffix: %q", st)
	}
}
