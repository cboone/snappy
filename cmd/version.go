package cmd

import "github.com/spf13/cobra"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of snappy",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Printf("snappy version %s\n", rootCmd.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
