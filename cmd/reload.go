package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/quay/lifecycle"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload projects",
	Long: `Reload one or more projects defined in the configuration file.
If no project is specified, all projects will be reloaded.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := lifecycle.NewManager(configFile, projectName, branch)
		return manager.ProcessProjects("reload")
	},
}

func init() {
	// Add persistent flags for config file and project name
	reloadCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file (required)")
	reloadCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of the project to operate on")
	reloadCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "branch to use for remote projects")

	// Mark config flag as required
	if err := reloadCmd.MarkPersistentFlagRequired("config"); err != nil {
		panic(err)
	}
}
