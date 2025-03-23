package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yarlson/quay/testutil"
)

type SmokeTestSuite struct {
	testutil.BaseSuite
}

func TestSmokeTestSuite(t *testing.T) {
	suite := &SmokeTestSuite{}
	testutil.RunSuite(t, suite)
}

func (s *SmokeTestSuite) TestLoadSimpleConfig() {
	// Create a simple test config file
	configContent := `
projects:
  - name: test-project
    path: /test/path
    compose_file: docker-compose.yml
`
	configPath := s.CreateTempFile("config.yaml", configContent)

	// Load the configuration
	cfg, err := LoadConfig(configPath)

	// Assert basic expectations
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cfg)
	assert.Len(s.T(), cfg.Projects, 1)
	assert.Equal(s.T(), "test-project", cfg.Projects[0].Name)
}
