package ingress

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SSLTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *SSLTestSuite) SetupSuite() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "ssl-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
}

func (s *SSLTestSuite) TearDownSuite() {
	_ = os.RemoveAll(s.tempDir)
}

func (s *SSLTestSuite) TestGenerateSSLCerts() {
	tests := []struct {
		name          string
		config        *Config
		expectedError bool
	}{
		{
			name: "ssl disabled",
			config: &Config{
				Hostname:   "example.com",
				SSLEnabled: false,
			},
			expectedError: false,
		},
		{
			name: "ssl enabled",
			config: &Config{
				Hostname:   "secure.example.com",
				SSLEnabled: true,
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Override the SSL directory for testing
			sslDir := filepath.Join(s.tempDir, "ssl")
			_ = os.Setenv("QUAY_SSL_DIR", sslDir)
			defer func() { _ = os.Unsetenv("QUAY_SSL_DIR") }()

			err := GenerateSSLCerts(tt.config)
			if tt.expectedError {
				assert.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			if tt.config.SSLEnabled {
				// Verify that certificate files were created
				certPath := filepath.Join(sslDir, tt.config.Hostname+".crt")
				keyPath := filepath.Join(sslDir, tt.config.Hostname+".key")

				assert.FileExists(s.T(), certPath, "Certificate file should exist")
				assert.FileExists(s.T(), keyPath, "Key file should exist")

				// Verify that paths were updated in the config
				assert.Equal(s.T(), certPath, tt.config.SSLCertPath)
				assert.Equal(s.T(), keyPath, tt.config.SSLKeyPath)
			}
		})
	}
}

func TestSSLSuite(t *testing.T) {
	suite.Run(t, new(SSLTestSuite))
}
