package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/yarlson/quay/config"
)

type LifecycleTestSuite struct {
	suite.Suite
	tempDir    string
	configFile string
	rootCmd    *cobra.Command
	nginxDir   string
}

func (s *LifecycleTestSuite) SetupSuite() {
	// Create temporary directory for test files
	var err error
	s.tempDir, err = os.MkdirTemp("", "lifecycle-test-*")
	require.NoError(s.T(), err)

	// Create temporary directory for Nginx config
	s.nginxDir = filepath.Join(s.tempDir, "nginx")
	err = os.MkdirAll(s.nginxDir, 0755)
	require.NoError(s.T(), err)

	// Create local project directory and compose file
	localProjectDir := filepath.Join(s.tempDir, "local-project")
	err = os.MkdirAll(localProjectDir, 0755)
	require.NoError(s.T(), err)

	composeContent := `
version: '3'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`
	err = os.WriteFile(filepath.Join(localProjectDir, "docker-compose.yml"), []byte(composeContent), 0644)
	require.NoError(s.T(), err)

	// Create remote project directory
	remoteProjectDir := filepath.Join(s.tempDir, "remote-project")
	err = os.MkdirAll(remoteProjectDir, 0755)
	require.NoError(s.T(), err)

	// Create test configuration file
	s.configFile = filepath.Join(s.tempDir, "config.yaml")
	configContent := `
projects:
  - name: local-project
    path: ./local-project
    compose_file: docker-compose.yml
    ports:
      - service: web
        host: "8080"
        container: "80"
    env:
      DEBUG: "true"

  - name: remote-project
    path: ./remote-project
    compose_file: docker-compose.yml
    ports:
      - service: api
        host: "3000"
        container: "3000"
    env:
      API_KEY: "test-key"
    remote:
      repo: "https://github.com/example/test-repo.git"
      branch: "main"
    ingress:
      enabled: true
      hostname: "remote.example.com"
      paths:
        - "/"
`
	err = os.WriteFile(s.configFile, []byte(configContent), 0644)
	require.NoError(s.T(), err)

	// Initialize the root command
	s.rootCmd = rootCmd

	// Set test mode to skip actual git operations
	err = os.Setenv("QUAY_TEST_MODE", "true")
	require.NoError(s.T(), err)

	// Set test mode for Nginx
	err = os.Setenv("NGINX_CONFIG_DIR", s.nginxDir)
	require.NoError(s.T(), err)
}

func (s *LifecycleTestSuite) TearDownSuite() {
	// Clean up temporary directory
	_ = os.RemoveAll(s.tempDir)
	_ = os.Unsetenv("QUAY_TEST_MODE")
	_ = os.Unsetenv("NGINX_CONFIG_DIR")
}

func (s *LifecycleTestSuite) TestLifecycleCommands() {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "start all projects",
			args:        []string{"lifecycle", "start", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "start specific project",
			args:        []string{"lifecycle", "start", "--config", s.configFile, "--project", "local-project"},
			expectError: false,
		},
		{
			name:        "start non-existent project",
			args:        []string{"lifecycle", "start", "--config", s.configFile, "--project", "non-existent"},
			expectError: true,
		},
		{
			name:        "stop all projects",
			args:        []string{"lifecycle", "stop", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "reload all projects",
			args:        []string{"lifecycle", "reload", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "status all projects",
			args:        []string{"lifecycle", "status", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "cleanup all projects",
			args:        []string{"lifecycle", "cleanup", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "remote project with branch override",
			args:        []string{"lifecycle", "start", "--config", s.configFile, "--project", "remote-project", "--branch", "develop"},
			expectError: false,
		},
		{
			name:        "missing config file",
			args:        []string{"lifecycle", "start"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Reset command flags and re-add them
			s.rootCmd.ResetFlags()
			s.rootCmd.ResetCommands()

			// Re-initialize the lifecycle command
			lifecycleCmd := &cobra.Command{
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

			// Re-add flags
			lifecycleCmd.PersistentFlags().StringVarP(&lifecycleConfigFile, "config", "c", "", "Path to the configuration file")
			lifecycleCmd.PersistentFlags().StringVarP(&lifecycleProject, "project", "p", "", "Name of the project to operate on (if not specified, operates on all projects)")
			lifecycleCmd.PersistentFlags().StringVarP(&lifecycleBranch, "branch", "b", "", "Override the branch for remote projects")

			// Re-add subcommands
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

			// Add a custom UnknownCommand handler
			lifecycleCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
				return fmt.Errorf("invalid command: %s", cmd.Name())
			})

			// Add a custom command validator
			lifecycleCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				if len(args) > 0 {
					return nil, cobra.ShellCompDirectiveError
				}
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			// Re-add the lifecycle command to the root command
			s.rootCmd.AddCommand(lifecycleCmd)

			// Execute the command
			s.rootCmd.SetArgs(tt.args)
			err := s.rootCmd.Execute()

			if tt.expectError {
				assert.Error(s.T(), err)
			} else {
				assert.NoError(s.T(), err)
			}
		})
	}
}

func TestLifecycleSuite(t *testing.T) {
	suite.Run(t, new(LifecycleTestSuite))
}
