package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"howett.net/plist"
)

// DefaultLabel is the launchd agent label for snappy.
const DefaultLabel = "com.cboone.snappy"

// PlistConfig holds the parameters for generating a launchd plist.
type PlistConfig struct {
	Label      string // launchd label (e.g., "com.cboone.snappy")
	BinaryPath string // resolved absolute path to the snappy binary
	LogDir     string // directory for stdout/stderr log files
	ConfigFile string // optional --config path; empty means use defaults
}

// Info describes the current state of the launchd agent.
type Info struct {
	Installed  bool
	Running    bool
	PID        int
	BinaryPath string // from the installed plist
	PlistPath  string
	Label      string
}

// generatedPlist defines the launchd agent plist structure for serialization
// via howett.net/plist. Using proper plist marshaling ensures XML-safe escaping
// of all values.
type generatedPlist struct {
	Label             string   `plist:"Label"`
	ProgramArguments  []string `plist:"ProgramArguments"`
	RunAtLoad         bool     `plist:"RunAtLoad"`
	KeepAlive         bool     `plist:"KeepAlive"`
	ProcessType       string   `plist:"ProcessType"`
	ThrottleInterval  int      `plist:"ThrottleInterval"`
	StandardOutPath   string   `plist:"StandardOutPath"`
	StandardErrorPath string   `plist:"StandardErrorPath"`
}

// PlistPath returns the expected plist file path for the given label:
// ~/Library/LaunchAgents/<label>.plist.
func PlistPath(label string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

// GeneratePlist serializes the launchd plist XML for the given configuration.
// Values are properly XML-escaped via howett.net/plist.
func GeneratePlist(cfg PlistConfig) ([]byte, error) {
	logPath := filepath.Join(cfg.LogDir, "snappy-service.log")
	args := []string{cfg.BinaryPath, "run"}
	if cfg.ConfigFile != "" {
		args = append(args, "--config", cfg.ConfigFile)
	}

	pl := generatedPlist{
		Label:             cfg.Label,
		ProgramArguments:  args,
		RunAtLoad:         true,
		KeepAlive:         true,
		ProcessType:       "Background",
		ThrottleInterval:  10,
		StandardOutPath:   logPath,
		StandardErrorPath: logPath,
	}

	data, err := plist.MarshalIndent(pl, plist.XMLFormat, "\t")
	if err != nil {
		return nil, fmt.Errorf("marshaling plist: %w", err)
	}
	return data, nil
}

// ResolveBinaryPath returns the canonical absolute path of the currently
// running executable, resolving any symlinks.
func ResolveBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks: %w", err)
	}
	return resolved, nil
}

// domainTarget returns the launchctl domain-target for the current user,
// e.g., "gui/502".
func domainTarget() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

// serviceTarget returns the launchctl service-target for the given label,
// e.g., "gui/502/com.cboone.snappy".
func serviceTarget(label string) string {
	return domainTarget() + "/" + label
}

// Install writes the plist file and loads it via launchctl bootstrap.
// If the agent is already loaded, it is unloaded first.
func Install(cfg PlistConfig) error {
	plistPath, err := PlistPath(cfg.Label)
	if err != nil {
		return err
	}

	// Unload any existing agent (ignore errors; it may not be loaded).
	_ = bootout(cfg.Label, plistPath)

	plistBytes, err := GeneratePlist(cfg)
	if err != nil {
		return err
	}

	launchAgentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0o755); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}

	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	if err := os.WriteFile(plistPath, plistBytes, 0o644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	//nolint:gosec // arguments are controlled, not user input
	cmd := exec.Command("launchctl", "bootstrap", domainTarget(), plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// Uninstall unloads the agent and removes the plist file.
func Uninstall(label string) error {
	plistPath, err := PlistPath(label)
	if err != nil {
		return err
	}

	if err := bootout(label, plistPath); err != nil {
		return err
	}

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}

	return nil
}

// Start sends a kickstart signal to the agent.
func Start(label string) error {
	//nolint:gosec // arguments are controlled
	cmd := exec.Command("launchctl", "kickstart", serviceTarget(label))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl kickstart: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Stop sends SIGTERM to the agent process.
func Stop(label string) error {
	//nolint:gosec // arguments are controlled
	cmd := exec.Command("launchctl", "kill", "SIGTERM", serviceTarget(label))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl kill: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Status queries launchctl for the current state of the agent.
func Status(label string) (*Info, error) {
	plistPath, err := PlistPath(label)
	if err != nil {
		return nil, err
	}

	st := &Info{
		Label:     label,
		PlistPath: plistPath,
	}

	// Check if plist file exists.
	if _, err := os.Stat(plistPath); err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return nil, fmt.Errorf("checking plist: %w", err)
	}
	st.Installed = true

	// Read binary path from the plist (best effort).
	st.BinaryPath = readBinaryFromPlist(plistPath)

	// Query launchctl for the running state.
	//nolint:gosec // arguments are controlled
	cmd := exec.Command("launchctl", "print", serviceTarget(label))
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		// Not loaded or not running is not an error for status;
		// launchctl returns non-zero when the service isn't registered.
		if isNotLoadedError(outStr) {
			return st, nil
		}
		return nil, fmt.Errorf("launchctl print: %s (%w)", outStr, err)
	}

	st.Running, st.PID = parseRuntimeFromPrint(outStr)

	return st, nil
}

// bootout unloads the agent. Returns nil if the agent was not loaded.
func bootout(label, plistPath string) error {
	// Try the modern service-target approach first.
	//nolint:gosec // arguments are controlled
	cmd := exec.Command("launchctl", "bootout", serviceTarget(label))
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	outStr := strings.TrimSpace(string(out))

	// If service-target bootout failed, try the legacy domain-target + plist path
	// approach, which works in some macOS versions where the service isn't
	// registered in the expected domain.
	if _, statErr := os.Stat(plistPath); statErr == nil {
		//nolint:gosec // arguments are controlled
		legacyCmd := exec.Command("launchctl", "bootout", domainTarget(), plistPath)
		legacyOut, legacyErr := legacyCmd.CombinedOutput()
		if legacyErr == nil {
			return nil
		}
		outStr = strings.TrimSpace(string(legacyOut))
		err = legacyErr
	}

	// "No such process" or "Could not find service" means it wasn't loaded.
	if isNotLoadedError(outStr) {
		return nil
	}

	return fmt.Errorf("launchctl bootout: %s (%w)", outStr, err)
}

// isNotLoadedError checks if a launchctl error message indicates the service
// simply wasn't loaded (not a real error for our purposes).
func isNotLoadedError(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "no such process") ||
		strings.Contains(lower, "could not find service")
}

// parsePIDFromPrint extracts the PID from `launchctl print` output.
// The line looks like: "pid = 12345" or "pid = -" (not running).
func parsePIDFromPrint(output string) int {
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if pidStr, ok := strings.CutPrefix(trimmed, "pid = "); ok {
			if pid, err := strconv.Atoi(pidStr); err == nil {
				return pid
			}
		}
	}
	return 0
}

// parseRuntimeFromPrint extracts runtime state from `launchctl print` output.
func parseRuntimeFromPrint(output string) (running bool, pid int) {
	pid = parsePIDFromPrint(output)
	return pid > 0, pid
}

// launchdPlist is the subset of a launchd plist we need for reading back
// the binary path and log path.
type launchdPlist struct {
	ProgramArguments []string `plist:"ProgramArguments"`
	StandardOutPath  string   `plist:"StandardOutPath"`
}

// readBinaryFromPlist reads the first ProgramArguments string from a plist
// file using proper plist parsing.
func readBinaryFromPlist(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var pl launchdPlist
	if _, err := plist.Unmarshal(data, &pl); err != nil {
		return ""
	}
	if len(pl.ProgramArguments) == 0 {
		return ""
	}
	return pl.ProgramArguments[0]
}

// LogPath returns the path to the service's stdout/stderr log file.
func LogPath(logDir string) string {
	return filepath.Join(logDir, "snappy-service.log")
}

// ReadLogPathFromPlist reads the StandardOutPath from an installed plist file.
// Returns an empty string if the file cannot be read or parsed.
func ReadLogPathFromPlist(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var pl launchdPlist
	if _, err := plist.Unmarshal(data, &pl); err != nil {
		return ""
	}
	return pl.StandardOutPath
}

// ErrNotInstalled is returned when a service operation requires an installed
// agent but none is found.
var ErrNotInstalled = errors.New("snappy service is not installed")
