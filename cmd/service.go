package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/service"
)

var serviceStatus = service.Status

var resolveServiceBinaryPath = service.ResolveBinaryPath

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the snappy background service (launchd)",
	Long: `Manage the snappy launchd agent that runs auto-snapshots in the background.

The service runs "snappy run" as a LaunchAgent, starting at login and
restarting automatically if it exits unexpectedly.`,
	Args: cobra.NoArgs,
	RunE: runServiceStatus,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and start the snappy launchd agent",
	Args:  cobra.NoArgs,
	RunE:  runServiceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop and remove the snappy launchd agent",
	Args:  cobra.NoArgs,
	RunE:  runServiceUninstall,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the snappy launchd agent",
	Args:  cobra.NoArgs,
	RunE:  runServiceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the snappy launchd agent",
	Args:  cobra.NoArgs,
	RunE:  runServiceStop,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the snappy launchd agent",
	Args:  cobra.NoArgs,
	RunE:  runServiceStatus,
}

var serviceLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Tail the snappy service log",
	Args:  cobra.NoArgs,
	RunE:  runServiceLog,
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceLogCmd)
	rootCmd.AddCommand(serviceCmd)
}

func runServiceInstall(cmd *cobra.Command, _ []string) error {
	binPath, err := resolveServiceBinaryPath()
	if err != nil {
		return err
	}

	cfg := config.Load()
	w := cmd.OutOrStdout()

	plistCfg := service.PlistConfig{
		Label:      service.DefaultLabel,
		BinaryPath: binPath,
		LogDir:     cfg.LogDir,
		ConfigFile: cfgFile,
	}

	if err := service.Install(plistCfg); err != nil {
		return err
	}

	plistPath, err := service.PlistPath(service.DefaultLabel)
	if err != nil {
		return fmt.Errorf("determining plist path: %w", err)
	}
	_, _ = fmt.Fprintf(w, "Installed and started snappy background service.\n")
	_, _ = fmt.Fprintf(w, "  Binary: %s\n", binPath)
	_, _ = fmt.Fprintf(w, "  Plist:  %s\n", plistPath)
	_, _ = fmt.Fprintf(w, "  Log:    %s\n", service.LogPath(cfg.LogDir))
	return nil
}

func runServiceUninstall(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	st, err := serviceStatus(service.DefaultLabel)
	if err != nil {
		return err
	}
	if !st.Installed {
		_, _ = fmt.Fprintln(w, "Service is not installed.")
		return nil
	}

	if err := service.Uninstall(service.DefaultLabel); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Stopped and removed snappy background service.")
	return nil
}

func runServiceStart(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	st, err := serviceStatus(service.DefaultLabel)
	if err != nil {
		return err
	}
	if !st.Installed {
		return service.ErrNotInstalled
	}
	if st.Running {
		_, _ = fmt.Fprintln(w, "Service is already running.")
		return nil
	}

	if err := service.Start(service.DefaultLabel); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Started snappy background service.")
	return nil
}

func runServiceStop(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	st, err := serviceStatus(service.DefaultLabel)
	if err != nil {
		return err
	}
	if !st.Installed {
		return service.ErrNotInstalled
	}
	if !st.Running {
		_, _ = fmt.Fprintln(w, "Service is not running.")
		return nil
	}

	if err := service.Stop(service.DefaultLabel); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Stopped snappy background service.")
	return nil
}

func runServiceStatus(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	st, err := serviceStatus(service.DefaultLabel)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Service: %s\n", st.Label)

	if !st.Installed {
		_, _ = fmt.Fprintf(w, "Status:  not installed\n")
		_, _ = fmt.Fprintf(w, "\nRun \"snappy service install\" to set up the background service.\n")
		return nil
	}

	_, _ = fmt.Fprintf(w, "Plist:   %s\n", st.PlistPath)

	if st.BinaryPath != "" {
		_, _ = fmt.Fprintf(w, "Binary:  %s\n", st.BinaryPath)

		// Check if the binary still exists.
		if _, statErr := os.Stat(st.BinaryPath); statErr != nil {
			_, _ = fmt.Fprintf(w, "         (warning: binary not found at this path)\n")
		}

		currentBinaryPath, resolveErr := resolveServiceBinaryPath()
		if resolveErr == nil && currentBinaryPath != "" && currentBinaryPath != st.BinaryPath {
			_, _ = fmt.Fprintf(w, "         (warning: installed service uses a different binary)\n")
			_, _ = fmt.Fprintf(w, "         (current snappy binary: %s)\n", currentBinaryPath)
			_, _ = fmt.Fprintf(w, "         (run \"snappy service install\" to update)\n")
		}
	}

	if st.Running {
		status := "running"
		if st.PID > 0 {
			status = fmt.Sprintf("running (PID %d)", st.PID)
		}
		_, _ = fmt.Fprintf(w, "Status:  %s\n", status)
	} else {
		_, _ = fmt.Fprintf(w, "Status:  stopped\n")
	}

	cfg := config.Load()
	_, _ = fmt.Fprintf(w, "Log:     %s\n", service.LogPath(cfg.LogDir))

	return nil
}

func runServiceLog(cmd *cobra.Command, _ []string) error {
	cfg := config.Load()
	logPath := service.LogPath(cfg.LogDir)

	if _, err := os.Stat(logPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service log not found at %s (is the service installed?)", logPath)
		}
		return err
	}

	// Replace the process with tail -f. This gives the user a live log view
	// that they can exit with Ctrl-C.
	tailPath, err := findTail()
	if err != nil {
		return err
	}
	return syscall.Exec(tailPath, []string{"tail", "-f", logPath}, os.Environ())
}

// findTail locates the tail binary.
var findTail = func() (string, error) {
	for _, p := range []string{"/usr/bin/tail", "/bin/tail"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("tail command not found")
}
