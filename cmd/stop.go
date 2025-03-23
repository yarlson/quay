package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/quay/lifecycle"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop projects",
	Long: `Stop one or more projects defined in the configuration file.
If no project is specified, all projects will be stopped.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := lifecycle.NewManager(configFile, projectName, branch)
		return manager.ProcessProjects("stop")
	},
}

func init() {
	// Add persistent flags for config file and project name
	stopCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file (required)")
	stopCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of the project to operate on")
	stopCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "branch to use for remote projects")

	// Mark config flag as required
	if err := stopCmd.MarkPersistentFlagRequired("config"); err != nil {
		panic(err)
	}
}
