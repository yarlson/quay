package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/quay/lifecycle"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start projects",
	Long: `Start one or more projects defined in the configuration file.
If no project is specified, all projects will be started.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := lifecycle.NewManager(configFile, projectName, branch)
		return manager.ProcessProjects("start")
	},
}

func init() {
	// Add persistent flags for config file and project name
	startCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file (required)")
	startCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of the project to operate on")
	startCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "branch to use for remote projects")

	// Mark config flag as required
	if err := startCmd.MarkPersistentFlagRequired("config"); err != nil {
		panic(err)
	}
}
