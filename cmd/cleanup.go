package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/quay/lifecycle"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up projects",
	Long: `Clean up one or more projects defined in the configuration file.
This will stop the projects and remove any temporary files.
If no project is specified, all projects will be cleaned up.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := lifecycle.NewManager(configFile, projectName, branch)
		return manager.ProcessProjects("cleanup")
	},
}

func init() {
	// Add persistent flags for config file and project name
	cleanupCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file (required)")
	cleanupCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of the project to operate on")
	cleanupCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "branch to use for remote projects")

	// Mark config flag as required
	if err := cleanupCmd.MarkPersistentFlagRequired("config"); err != nil {
		panic(err)
	}
}
