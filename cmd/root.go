package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:           "snappy",
		Short:         "Automatically increase your Time Machine snapshot frequency",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string on the root command.
func SetVersion(v string) {
	rootCmd.Version = v
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

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Missing config file is fine; surface unexpected errors (e.g., syntax).
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Warning: config file error: %v\n", err)
		}
	}
}
