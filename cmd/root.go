package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yarlson/quay/config"
)

var (
	configFile string
	project    string
	branch     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "quay",
	Short: "A CLI tool for managing Docker Compose operations with port mapping support",
	Long: `Quay is a CLI tool that extends Docker Compose functionality with support for
dynamic port mapping overrides. It allows you to modify port mappings without
editing the compose file directly.`,
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
	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVarP(&project, "project", "p", "", "Name of the project to operate on (if not specified, operates on all projects)")
	rootCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "Override the branch for remote projects")

	// Add commands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cleanupCmd)
}

// Common function to validate and process projects
func processProjects(cmdName string) error {
	// Check if config file is provided
	if configFile == "" {
		return fmt.Errorf("config file is required")
	}

	// Load the configuration file
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Filter projects if project name is specified
	var projects []config.Project
	if project != "" {
		found := false
		for _, p := range cfg.Projects {
			if p.Name == project {
				projects = append(projects, p)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("project %s not found in configuration", project)
		}
	} else {
		projects = cfg.Projects
	}

	// Process each project
	for _, project := range projects {
		// Override branch if specified
		if branch != "" && project.Remote != nil {
			project.Remote.Branch = branch
		}

		// Handle remote project provisioning
		if project.Remote != nil {
			if err := handleRemoteProject(&project); err != nil {
				return fmt.Errorf("failed to handle remote project %s: %w", project.Name, err)
			}
		}

		// Execute the requested command
		var err error
		switch cmdName {
		case "start":
			err = startProject(&project)
		case "stop":
			err = stopProject(&project)
		case "reload":
			err = reloadProject(&project)
		case "status":
			err = statusProject(&project)
		case "cleanup":
			err = cleanupProject(&project)
		default:
			return fmt.Errorf("invalid command: %s", cmdName)
		}

		if err != nil {
			return fmt.Errorf("failed to %s project %s: %w", cmdName, project.Name, err)
		}
	}

	return nil
}
