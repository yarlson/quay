package cmd

import (
	"github.com/spf13/cobra"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload projects",
	Long: `Reload one or more projects defined in the configuration file.
If no project is specified, all projects will be reloaded.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return processProjects("reload")
	},
}
