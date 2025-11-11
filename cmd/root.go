package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "issue-miner",
	Short: "Analyze GitHub issues",
	Long:  "issue-miner: metrics and graphs for GitHub issues (gh extension)",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(fetchCmd)
}
