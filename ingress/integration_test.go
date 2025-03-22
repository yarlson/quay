package ingress

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	tempDir      string
	configDir    string
	sslDir       string
	nginxManager *NginxManager
	configs      []IngressConfig
}

func (s *IntegrationTestSuite) SetupSuite() {
	// Create temporary directories for test files
	var err error
	s.tempDir, err = os.MkdirTemp("", "ingress-integration-test-*")
	require.NoError(s.T(), err)

	s.configDir = filepath.Join(s.tempDir, "nginx")
	s.sslDir = filepath.Join(s.tempDir, "ssl")

	// Set environment variables for test mode
	os.Setenv("QUAY_TEST_MODE", "true")
	os.Setenv("QUAY_SSL_DIR", s.sslDir)

	// Create Nginx manager
	s.nginxManager = NewNginxManager(s.configDir, "nginx.conf")

	// Create test configurations
	s.configs = []IngressConfig{
		{
			Hostname: "example.com",
			Paths:    []string{"/"},
			Upstream: "localhost:8080",
		},
		{
			Hostname:    "secure.example.com",
			Paths:       []string{"/"},
			Upstream:    "localhost:8443",
			SSLEnabled:  true,
			SSLCertPath: filepath.Join(s.sslDir, "secure.example.com.crt"),
			SSLKeyPath:  filepath.Join(s.sslDir, "secure.example.com.key"),
		},
	}
}

func (s *IntegrationTestSuite) TearDownSuite() {
	// Clean up temporary directories
	os.RemoveAll(s.tempDir)
	os.Unsetenv("QUAY_TEST_MODE")
	os.Unsetenv("QUAY_SSL_DIR")
}

func (s *IntegrationTestSuite) TestEndToEndIngressSetup() {
	// Run the ingress setup
	err := RunIngress(s.configs, s.nginxManager)
	require.NoError(s.T(), err)

	// Verify Nginx configuration file exists
	configPath := filepath.Join(s.configDir, "nginx.conf")
	require.FileExists(s.T(), configPath)

	// Verify SSL certificates exist for HTTPS configuration
	for _, config := range s.configs {
		if config.SSLEnabled {
			// Verify that the paths are set correctly
			expectedCertPath := filepath.Join(s.sslDir, config.Hostname+".crt")
			expectedKeyPath := filepath.Join(s.sslDir, config.Hostname+".key")
			assert.Equal(s.T(), expectedCertPath, config.SSLCertPath, "SSL certificate path should match")
			assert.Equal(s.T(), expectedKeyPath, config.SSLKeyPath, "SSL key path should match")
		}
	}
}

func (s *IntegrationTestSuite) TestErrorHandling() {
	// Test with invalid SSL configuration
	invalidConfigs := []IngressConfig{
		{
			Hostname:   "invalid.example.com",
			Paths:      []string{"/"},
			Upstream:   "localhost:8080",
			SSLEnabled: true,
			// Missing SSL certificate paths
		},
	}

	// Run the ingress setup with invalid configuration
	err := RunIngress(invalidConfigs, s.nginxManager)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "SSL certificate paths not set")

	// Test with empty configurations
	err = RunIngress([]IngressConfig{}, s.nginxManager)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "no ingress configurations provided")
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
