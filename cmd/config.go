package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or manage snappy configuration",
	Args:  cobra.NoArgs,
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file",
	Args:  cobra.NoArgs,
	RunE:  runConfigInit,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, _ []string) error {
	cfg := config.Load()
	return config.FormatConfig(cmd.OutOrStdout(), cfg, viper.ConfigFileUsed())
}

func runConfigInit(cmd *cobra.Command, _ []string) error {
	target := cfgFile
	if target == "" {
		p, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}
		target = p
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("config file already exists: %s", target)
		}
		return fmt.Errorf("creating config file: %w", err)
	}

	if err := config.WriteDefaultConfig(f); err != nil {
		_ = f.Close()
		_ = os.Remove(target)
		return fmt.Errorf("writing config: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("closing config file: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Config file created: %s\n", target)
	return nil
}
