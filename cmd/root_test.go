package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

func (s *LifecycleTestSuite) TestCommands() {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "start all projects",
			args:        []string{"start", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "start specific project",
			args:        []string{"start", "--config", s.configFile, "--project", "local-project"},
			expectError: false,
		},
		{
			name:        "start non-existent project",
			args:        []string{"start", "--config", s.configFile, "--project", "non-existent"},
			expectError: true,
		},
		{
			name:        "stop all projects",
			args:        []string{"stop", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "reload all projects",
			args:        []string{"reload", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "status all projects",
			args:        []string{"status", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "cleanup all projects",
			args:        []string{"cleanup", "--config", s.configFile},
			expectError: false,
		},
		{
			name:        "remote project with branch override",
			args:        []string{"start", "--config", s.configFile, "--project", "remote-project", "--branch", "develop"},
			expectError: false,
		},
		{
			name:        "missing config file",
			args:        []string{"start"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Reset command flags and re-add them
			s.rootCmd.ResetFlags()
			s.rootCmd.ResetCommands()

			// Re-add the commands
			s.rootCmd.AddCommand(startCmd)
			s.rootCmd.AddCommand(stopCmd)
			s.rootCmd.AddCommand(reloadCmd)
			s.rootCmd.AddCommand(statusCmd)
			s.rootCmd.AddCommand(cleanupCmd)

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
