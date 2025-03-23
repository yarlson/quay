package compose

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

func (s *SmokeTestSuite) TestLoadSimpleComposeFile() {
	// Create a simple test compose file
	composeContent := `
version: "3.8"
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`
	composePath := s.CreateTempFile("docker-compose.yml", composeContent)

	// Load the compose file
	project, err := LoadComposeFile(composePath)

	// Assert basic expectations
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), project)
	assert.Len(s.T(), project.Services, 1)
	assert.Equal(s.T(), "web", project.Services[0].Name)
	assert.Equal(s.T(), "nginx:latest", project.Services[0].Image)
}
