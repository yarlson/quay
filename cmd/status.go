package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/quay/lifecycle"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get project status",
	Long: `Get the status of one or more projects defined in the configuration file.
If no project is specified, status of all projects will be shown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := lifecycle.NewManager(configFile, projectName, branch)
		return manager.ProcessProjects("status")
	},
}

func init() {
	// Add persistent flags for config file and project name
	statusCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file (required)")
	statusCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of the project to operate on")
	statusCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "branch to use for remote projects")

	// Mark config flag as required
	if err := statusCmd.MarkPersistentFlagRequired("config"); err != nil {
		panic(err)
	}
}
