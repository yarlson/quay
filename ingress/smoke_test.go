package ingress

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

func (s *SmokeTestSuite) TestGenerateSimpleNginxConfig() {
	// Create a simple ingress configuration
	configs := []Config{
		{
			Hostname:   "example.com",
			Paths:      []string{"/"},
			Upstream:   "localhost:8080",
			SSLEnabled: false,
		},
	}

	// Generate Nginx configuration
	config, err := GenerateNginxConfig(configs)

	// Assert basic expectations
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), config)
	assert.Contains(s.T(), config, "server_name example.com;")
	assert.Contains(s.T(), config, "proxy_pass http://localhost:8080;")
}
