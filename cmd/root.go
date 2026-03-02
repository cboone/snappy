// Package cmd provides the Cobra root command that launches the Snappy TUI.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/tui"
)

var (
	cfgFile string
	version string
	rootCmd = &cobra.Command{
		Use:           "snappy",
		Short:         "Automatically increase your Time Machine snapshot frequency",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runTUI,
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string on the root command.
func SetVersion(v string) {
	rootCmd.Version = v
	version = v
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/snappy/config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "snappy"))
			viper.SetConfigName("config")
			viper.SetConfigType("yaml")
		}
	}

	viper.SetEnvPrefix("SNAPPY")
	viper.AutomaticEnv()
	config.SetDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Warning: config file error: %v\n", err)
		}
	}
}

func runTUI(_ *cobra.Command, _ []string) error {
	if _, err := exec.LookPath("tmutil"); err != nil {
		return fmt.Errorf("tmutil not found: this tool requires macOS with Time Machine support")
	}

	cfg := config.Load()

	// Resolve default log directory
	if cfg.LogDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cfg.LogDir = filepath.Join(home, ".local", "share", "snappy")
		}
	}

	log := logger.New(cfg.LogDir, 50)
	defer log.Close()

	runner := platform.OSRunner{}
	startupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// One-time startup: discover APFS volume and check TM status
	apfsVolume, err := platform.FindAPFSVolume(startupCtx, runner, cfg.MountPoint)
	if err != nil {
		log.Log(logger.Startup, fmt.Sprintf("Warning: failed to discover APFS volume for %s: %v", cfg.MountPoint, err))
	}
	tmStatus := platform.CheckStatus(startupCtx, runner)

	log.Log(logger.Startup, fmt.Sprintf("snappy v%s | volume=%s | refresh=%ds",
		version, cfg.MountPoint, int(cfg.RefreshInterval.Seconds())))
	log.Log(logger.Startup, fmt.Sprintf("auto-snapshot=%v | every %ds | thin >%ds to %ds",
		cfg.AutoEnabled, int(cfg.AutoSnapshotInterval.Seconds()),
		int(cfg.ThinAgeThreshold.Seconds()), int(cfg.ThinCadence.Seconds())))

	model := tui.NewModel(cfg, runner, log, apfsVolume, tmStatus, version)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
