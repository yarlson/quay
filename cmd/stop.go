package cmd

import (
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop projects",
	Long: `Stop one or more projects defined in the configuration file.
If no project is specified, all projects will be stopped.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return processProjects("stop")
	},
}
