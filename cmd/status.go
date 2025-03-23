package cmd

import (
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get project status",
	Long: `Get the status of one or more projects defined in the configuration file.
If no project is specified, status of all projects will be shown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return processProjects("status")
	},
}
