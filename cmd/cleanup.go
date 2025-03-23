package cmd

import (
	"github.com/spf13/cobra"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up projects",
	Long: `Clean up one or more projects defined in the configuration file.
This will stop the projects and remove any temporary files.
If no project is specified, all projects will be cleaned up.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return processProjects("cleanup")
	},
}
