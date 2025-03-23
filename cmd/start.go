package cmd

import (
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start projects",
	Long: `Start one or more projects defined in the configuration file.
If no project is specified, all projects will be started.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return processProjects("start")
	},
}
