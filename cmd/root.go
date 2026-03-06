// Package cmd provides the Cobra root command that launches the Snappy TUI.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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

	helpDefault := "~/.config/snappy/config.yaml"
	if p, err := config.DefaultConfigPath(); err == nil {
		if home, err := os.UserHomeDir(); err == nil {
			sep := string(os.PathSeparator)
			if strings.HasPrefix(p, home+sep) {
				rel := strings.TrimPrefix(p, home)
				helpDefault = "~" + rel
			} else {
				helpDefault = p
			}
		} else {
			helpDefault = p
		}
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		fmt.Sprintf("config file (default: %s)", helpDefault))
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		defaultPath, err := config.DefaultConfigPath()
		if err == nil {
			viper.AddConfigPath(filepath.Dir(defaultPath))
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

	log := logger.New(logger.Options{
		LogDir:     cfg.LogDir,
		MaxEntries: 50,
		MaxSize:    cfg.LogMaxSize,
		MaxFiles:   cfg.LogMaxFiles,
	})
	defer log.Close()

	runner := platform.OSRunner{}
	startupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// One-time startup: discover APFS volume and check TM status
	apfsVolume, err := platform.FindAPFSVolume(startupCtx, runner, config.DefaultMount)
	if err != nil {
		log.Log(logger.Startup, fmt.Sprintf("Warning: failed to discover APFS volume for %s: %v", config.DefaultMount, err))
	}
	tmStatus := platform.CheckStatus(startupCtx, runner)

	volumeName := config.DefaultMount
	var apfsContainer string
	if mountInfo, mountErr := platform.GetMountInfo(startupCtx, runner, config.DefaultMount); mountErr == nil {
		if mountInfo.VolumeName != "" {
			volumeName = mountInfo.VolumeName
		}
		apfsContainer = mountInfo.APFSContainerReference
	}

	log.Log(logger.Startup, startupSummary(version, volumeName, cfg.RefreshInterval))
	if apfsVolume != "" {
		log.Log(logger.Startup, fmt.Sprintf("apfs-volume=%s", apfsVolume))
	}
	if apfsContainer != "" {
		log.Log(logger.Startup, fmt.Sprintf("apfs-container=%s", apfsContainer))
	}
	log.Log(logger.Startup, fmt.Sprintf("auto-snapshot=%v | every %ds | thin >%ds to %ds",
		cfg.AutoEnabled, int(cfg.AutoSnapshotInterval.Seconds()),
		int(cfg.ThinAgeThreshold.Seconds()), int(cfg.ThinCadence.Seconds())))

	model := tui.NewModel(cfg, runner, log, apfsVolume, apfsContainer, tmStatus, volumeName, version)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}

func startupSummary(version, volumeName string, refreshInterval time.Duration) string {
	return fmt.Sprintf("snappy %s | volume=%s | refresh=%ds",
		version, volumeName, int(refreshInterval.Seconds()))
}
