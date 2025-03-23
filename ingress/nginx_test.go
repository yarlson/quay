package ingress

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type NginxTestSuite struct {
	suite.Suite
	tempDir    string
	configDir  string
	configFile string
	manager    *NginxManager
}

func (s *NginxTestSuite) SetupSuite() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "nginx-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir

	// Set up test directories and files
	s.configDir = filepath.Join(s.tempDir, "nginx")
	s.configFile = "nginx.conf"
	s.manager = NewNginxManager(s.configDir, s.configFile)
}

func (s *NginxTestSuite) TearDownSuite() {
	_ = os.RemoveAll(s.tempDir)
}

func (s *NginxTestSuite) TestWriteConfig() {
	// Test writing a valid configuration
	config := `events {
    worker_connections 1024;
}

http {
    server {
        listen 80;
        server_name example.com;
    }
}`

	err := s.manager.WriteConfig(config)
	require.NoError(s.T(), err)

	// Verify that the config file was created
	configPath := filepath.Join(s.configDir, s.configFile)
	assert.FileExists(s.T(), configPath, "Configuration file should exist")

	// Verify the file contents
	content, err := os.ReadFile(configPath)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), config, string(content))

	// Test writing to an invalid directory
	invalidManager := NewNginxManager("/invalid/path", "nginx.conf")
	err = invalidManager.WriteConfig(config)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to create config directory")
}

func (s *NginxTestSuite) TestNginxCommands() {
	// Test Nginx status check without container name
	err := s.manager.CheckNginxStatus()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "QUAY_NGINX_CONTAINER environment variable not set")

	// Test Nginx reload without container name
	err = s.manager.ReloadNginx()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "QUAY_NGINX_CONTAINER environment variable not set")

	// Test Nginx restart without container name
	err = s.manager.RestartNginx()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "QUAY_NGINX_CONTAINER environment variable not set")
}

func (s *NginxTestSuite) TestDockerContainerMode() {
	// Test with a mock Docker container
	_ = os.Setenv("QUAY_NGINX_CONTAINER", "test-nginx")
	defer func() { _ = os.Unsetenv("QUAY_NGINX_CONTAINER") }()

	// Test Nginx status check in Docker mode
	err := s.manager.CheckNginxStatus()
	// We expect an error since the container doesn't exist
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to check Nginx container status")

	// Test Nginx reload in Docker mode
	err = s.manager.ReloadNginx()
	// We expect an error since the container doesn't exist
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to reload Nginx in container")

	// Test Nginx restart in Docker mode
	err = s.manager.RestartNginx()
	// We expect an error since the container doesn't exist
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to restart Nginx container")
}

func TestNginxSuite(t *testing.T) {
	suite.Run(t, new(NginxTestSuite))
}
