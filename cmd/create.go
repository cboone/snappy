package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/platform"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new local Time Machine snapshot",
	Args:  cobra.NoArgs,
	RunE:  runCreate,
}

func init() {
	createCmd.Flags().Bool("json", false, "output in JSON format")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, _ []string) error {
	if err := requireTmutil(); err != nil {
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	runner := newRunner()

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
	defer cancel()

	date, err := platform.CreateSnapshot(ctx, runner)
	if err != nil {
		return err
	}

	if jsonOut {
		return writeJSON(cmd.OutOrStdout(), struct {
			Date string `json:"date"`
		}{Date: date})
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Snapshot created: %s\n", date)
	return err
}
