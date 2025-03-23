package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yarlson/quay/compose"
	"github.com/yarlson/quay/config"
	"github.com/yarlson/quay/ingress"
)

var (
	lifecycleConfigFile string
	lifecycleProject    string
	lifecycleBranch     string
)

// lifecycleCmd represents the lifecycle command
var lifecycleCmd = &cobra.Command{
	Use:   "lifecycle",
	Short: "Manage project lifecycles with remote project support",
	Long: `Manage project lifecycles with support for remote project provisioning.
Supported commands: start, stop, reload, status, cleanup

Example:
  quay lifecycle start --config config.yaml
  quay lifecycle stop --config config.yaml --project web-project
  quay lifecycle status --config config.yaml
  quay lifecycle cleanup --config config.yaml`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "lifecycle" {
			return nil // Skip validation for the root command
		}
		if lifecycleConfigFile == "" {
			return fmt.Errorf("config file is required")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lifecycleCmd)

	// Add persistent flags
	lifecycleCmd.PersistentFlags().StringVarP(&lifecycleConfigFile, "config", "c", "", "Path to the configuration file")
	lifecycleCmd.PersistentFlags().StringVarP(&lifecycleProject, "project", "p", "", "Name of the project to operate on (if not specified, operates on all projects)")
	lifecycleCmd.PersistentFlags().StringVarP(&lifecycleBranch, "branch", "b", "", "Override the branch for remote projects")

	// Set up command validation
	lifecycleCmd.SilenceErrors = true
	lifecycleCmd.SilenceUsage = true
	lifecycleCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q", args[0])
		}
		cmd.Root().HelpFunc()(cmd, args)
		return nil
	}

	// Add subcommands
	validCommands := map[string]bool{
		"start":   true,
		"stop":    true,
		"reload":  true,
		"status":  true,
		"cleanup": true,
	}

	for cmdName := range validCommands {
		cmdName := cmdName // Create a new variable to avoid closure issues
		subCmd := &cobra.Command{
			Use:   cmdName,
			Short: fmt.Sprintf("%s projects", cmdName),
			RunE: func(cmd *cobra.Command, args []string) error {
				// Check if config file is provided
				if lifecycleConfigFile == "" {
					return fmt.Errorf("config file is required")
				}

				// Load the configuration file
				cfg, err := config.LoadConfig(lifecycleConfigFile)
				if err != nil {
					return fmt.Errorf("failed to load configuration: %w", err)
				}

				// Filter projects if project name is specified
				var projects []config.Project
				if lifecycleProject != "" {
					found := false
					for _, p := range cfg.Projects {
						if p.Name == lifecycleProject {
							projects = append(projects, p)
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("project %s not found in configuration", lifecycleProject)
					}
				} else {
					projects = cfg.Projects
				}

				// Process each project
				for _, project := range projects {
					// Override branch if specified
					if lifecycleBranch != "" && project.Remote != nil {
						project.Remote.Branch = lifecycleBranch
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
			},
		}
		lifecycleCmd.AddCommand(subCmd)
	}
}

// handleRemoteProject handles remote project provisioning
func handleRemoteProject(project *config.Project) error {
	// Create project directory if it doesn't exist
	if err := os.MkdirAll(project.Path, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Skip git operations in test mode
	if os.Getenv("QUAY_TEST_MODE") == "true" {
		// Create a dummy compose file for testing
		composeContent := `
version: '3'
services:
  api:
    image: nginx:latest
    ports:
      - "3000:3000"
`
		err := os.WriteFile(filepath.Join(project.Path, "docker-compose.yml"), []byte(composeContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to create test compose file: %w", err)
		}
		return nil
	}

	// Check if the directory is empty
	entries, err := os.ReadDir(project.Path)
	if err != nil {
		return fmt.Errorf("failed to read project directory: %w", err)
	}

	// If directory is empty, clone the repository
	if len(entries) == 0 {
		cmd := exec.Command("git", "clone", "-b", project.Remote.Branch, project.Remote.Repo, project.Path)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repository: %s: %w", string(output), err)
		}
	} else {
		// If directory is not empty, pull latest changes
		cmd := exec.Command("git", "-C", project.Path, "pull", "origin", project.Remote.Branch)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %s: %w", string(output), err)
		}
	}

	return nil
}

// startProject starts a project
func startProject(project *config.Project) error {
	// Convert port mappings
	var mappings []compose.PortMapping
	for _, p := range project.Ports {
		mappings = append(mappings, compose.PortMapping{
			Service:   p.Service,
			Host:      p.Host,
			Container: p.Container,
		})
	}

	// Start the project
	return compose.RunComposeOperation(project.ComposeFile, mappings, []string{"up", "-d"})
}

// stopProject stops a project
func stopProject(project *config.Project) error {
	return compose.RunComposeOperation(project.ComposeFile, nil, []string{"down"})
}

// reloadProject reloads a project
func reloadProject(project *config.Project) error {
	// First stop the project
	if err := stopProject(project); err != nil {
		return err
	}

	// Then start it again
	return startProject(project)
}

// statusProject gets the status of a project
func statusProject(project *config.Project) error {
	return compose.RunComposeOperation(project.ComposeFile, nil, []string{"ps"})
}

// cleanupProject cleans up a project
func cleanupProject(project *config.Project) error {
	// Stop the project
	if err := stopProject(project); err != nil {
		return err
	}

	// Remove the project directory if it's a remote project
	if project.Remote != nil {
		if err := os.RemoveAll(project.Path); err != nil {
			return fmt.Errorf("failed to remove project directory: %w", err)
		}
	}

	// Clean up ingress configuration if enabled
	if project.Ingress != nil && project.Ingress.Enabled {
		nginxConfigDir := os.Getenv("NGINX_CONFIG_DIR")
		if nginxConfigDir == "" {
			nginxConfigDir = "/etc/nginx/conf.d"
		}
		nginxManager := ingress.NewNginxManager(nginxConfigDir, "quay.conf")
		if err := nginxManager.WriteConfig(""); err != nil {
			return fmt.Errorf("failed to clean up ingress configuration: %w", err)
		}
	}

	return nil
}
