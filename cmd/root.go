package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yarlson/quay/lifecycle"
)

var (
	configFile  string
	projectName string
	branch      string
)

// setupFlags adds persistent flags to the root command
func setupFlags(cmd *cobra.Command) {
	// Add persistent flags for config file and project name
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file (required)")
	cmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of the project to operate on")
	cmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "branch to use for remote projects")

	// Mark config flag as required
	if err := cmd.MarkPersistentFlagRequired("config"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking config flag as required: %v\n", err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "quay",
	Short: "A CLI tool for managing multiple Docker projects",
	Long: `Quay is a CLI tool that helps you manage multiple Docker projects locally.
It provides commands to start, stop, reload, and check the status of your projects.
You can also manage remote projects by specifying their repository URLs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if config file is provided
		if configFile == "" {
			return fmt.Errorf("config file is required")
		}

		// Create a lifecycle manager
		manager := lifecycle.NewManager(configFile, projectName, branch)

		// Process projects based on the command
		return manager.ProcessProjects("start")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add commands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cleanupCmd)

	// Set up flags
	setupFlags(rootCmd)
}
