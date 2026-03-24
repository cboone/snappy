package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestIsAlreadyBootstrappedError(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"Bootstrap failed: 125: Domain does not support specified action", true},
		{"domain does not support specified action", true},
		{"Service is already loaded", true},
		{"Bootstrap failed: 5: Input/output error", false},
		{"Operation not permitted", false},
		{"No such process", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isAlreadyBootstrappedError(tt.output)
		if got != tt.want {
			t.Errorf("isAlreadyBootstrappedError(%q) = %v, want %v", tt.output, got, tt.want)
		}
	}
}

func TestIsDomainNotSupportedError(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"Boot-out failed: 125: Domain does not support specified action", true},
		{"Bootstrap failed: 125: Domain does not support specified action", true},
		{"domain does not support specified action", true},
		{"Domain Does Not Support Specified Action", true},
		{"No such process", false},
		{"Could not find service", false},
		{"Bootstrap failed: 5: Input/output error", false},
		{"Operation not permitted", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isDomainNotSupportedError(tt.output)
		if got != tt.want {
			t.Errorf("isDomainNotSupportedError(%q) = %v, want %v", tt.output, got, tt.want)
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

type launchctlStep struct {
	args []string
	out  string
	err  error
}

func setTestHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
}

func writeInstalledPlist(t *testing.T, label string) string {
	t.Helper()
	plistPath, err := PlistPath(label)
	if err != nil {
		t.Fatalf("PlistPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(plistPath, []byte("plist"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return plistPath
}

func mockLaunchctl(t *testing.T, steps []launchctlStep) {
	t.Helper()
	orig := runLaunchctl
	idx := 0
	runLaunchctl = func(args ...string) ([]byte, error) {
		t.Helper()
		if idx >= len(steps) {
			t.Fatalf("unexpected launchctl call %v", args)
		}
		step := steps[idx]
		idx++
		if !reflect.DeepEqual(args, step.args) {
			t.Fatalf("launchctl call %d = %v, want %v", idx, args, step.args)
		}
		return []byte(step.out), step.err
	}
	t.Cleanup(func() {
		runLaunchctl = orig
		if idx != len(steps) {
			t.Errorf("launchctl calls = %d, want %d", idx, len(steps))
		}
	})
}

func TestStartFallsBackToBootstrapWhenKickstartServiceMissing(t *testing.T) {
	setTestHome(t)
	target := serviceTarget(DefaultLabel)
	plistPath, err := PlistPath(DefaultLabel)
	if err != nil {
		t.Fatalf("PlistPath() error = %v", err)
	}

	mockLaunchctl(t, []launchctlStep{
		{args: []string{"enable", target}},
		{
			args: []string{"kickstart", target},
			out:  "Could not find service \"com.cboone.snappy\" in domain for uid: 502",
			err:  errors.New("exit status 3"),
		},
		{args: []string{"bootstrap", domainTarget(), plistPath}},
	})

	if err := Start(DefaultLabel); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestStartReturnsKickstartErrorWhenServiceExistsButCannotStart(t *testing.T) {
	setTestHome(t)
	target := serviceTarget(DefaultLabel)

	mockLaunchctl(t, []launchctlStep{
		{args: []string{"enable", target}},
		{
			args: []string{"kickstart", target},
			out:  "Operation not permitted",
			err:  errors.New("exit status 1"),
		},
	})

	err := Start(DefaultLabel)
	if err == nil {
		t.Fatal("Start() should return error when kickstart fails with a non-fallback error")
	}
	if !strings.Contains(err.Error(), "launchctl kickstart") {
		t.Fatalf("Start() error = %v, want launchctl kickstart context", err)
	}
}

func TestStopFallsBackToLegacyBootoutWhenServiceTargetFails(t *testing.T) {
	setTestHome(t)
	target := serviceTarget(DefaultLabel)
	plistPath := writeInstalledPlist(t, DefaultLabel)

	mockLaunchctl(t, []launchctlStep{
		{args: []string{"disable", target}},
		{
			args: []string{"bootout", target},
			out:  "Boot-out failed: 125: Domain does not support specified action",
			err:  errors.New("exit status 125"),
		},
		{args: []string{"bootout", domainTarget(), plistPath}},
	})

	if err := Stop(DefaultLabel); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestStopFallsBackToUnloadWhenBootoutVariantsFail(t *testing.T) {
	setTestHome(t)
	target := serviceTarget(DefaultLabel)
	plistPath := writeInstalledPlist(t, DefaultLabel)

	mockLaunchctl(t, []launchctlStep{
		{args: []string{"disable", target}},
		{
			args: []string{"bootout", target},
			out:  "Boot-out failed: 5: Input/output error",
			err:  errors.New("exit status 5"),
		},
		{
			args: []string{"bootout", domainTarget(), plistPath},
			out:  "Boot-out failed: 5: Input/output error",
			err:  errors.New("exit status 5"),
		},
		{args: []string{"unload", plistPath}},
	})

	if err := Stop(DefaultLabel); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestStopTreatsNotLoadedAsSuccess(t *testing.T) {
	setTestHome(t)
	target := serviceTarget(DefaultLabel)

	mockLaunchctl(t, []launchctlStep{
		{args: []string{"disable", target}},
		{
			args: []string{"bootout", target},
			out:  "Could not find service \"com.cboone.snappy\" in domain for uid: 502",
			err:  errors.New("exit status 3"),
		},
	})

	if err := Stop(DefaultLabel); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
