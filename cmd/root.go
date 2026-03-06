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

// usageTemplate is a copy of cobra's default usage template with one change:
// the Usage section prints a single combined line instead of separate lines
// for [flags] and [command] when a command is both runnable and has subcommands.
const usageTemplate = `Usage:{{if or .Runnable .HasAvailableSubCommands}}
  {{.CommandPath}}{{if .HasAvailableSubCommands}} [command]{{end}}{{if .HasAvailableFlags}} [flags]{{end}}{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

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

	// Override the default usage template so that runnable commands with
	// subcommands show a single combined usage line instead of two separate
	// lines ("snappy [flags]" and "snappy [command]").
	rootCmd.SetUsageTemplate(usageTemplate)
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
	apfsVolume, err := platform.FindAPFSVolume(startupCtx, runner, cfg.MountPoint)
	if err != nil {
		log.Log(logger.Startup, fmt.Sprintf("Warning: failed to discover APFS volume for %s: %v", cfg.MountPoint, err))
	}
	tmStatus := platform.CheckStatus(startupCtx, runner)

	volumeName, err := platform.GetVolumeName(startupCtx, runner, cfg.MountPoint)
	if err != nil || volumeName == "" {
		volumeName = cfg.MountPoint
	}

	log.Log(logger.Startup, fmt.Sprintf("snappy %s | volume=%s | refresh=%ds",
		version, volumeName, int(cfg.RefreshInterval.Seconds())))
	if apfsVolume != "" {
		log.Log(logger.Startup, fmt.Sprintf("apfs-volume=%s", apfsVolume))
	}
	log.Log(logger.Startup, fmt.Sprintf("auto-snapshot=%v | every %ds | thin >%ds to %ds",
		cfg.AutoEnabled, int(cfg.AutoSnapshotInterval.Seconds()),
		int(cfg.ThinAgeThreshold.Seconds()), int(cfg.ThinCadence.Seconds())))

	model := tui.NewModel(cfg, runner, log, apfsVolume, tmStatus, volumeName, version)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
