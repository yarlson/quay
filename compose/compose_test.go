package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComposeTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *ComposeTestSuite) SetupSuite() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
}

func (s *ComposeTestSuite) TearDownSuite() {
	os.RemoveAll(s.tempDir)
}

func (s *ComposeTestSuite) createTestComposeFile(content string) string {
	path := filepath.Join(s.tempDir, "docker-compose.yml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(s.T(), err)
	return path
}

func (s *ComposeTestSuite) TestLoadComposeFile() {
	// Test valid compose file
	validContent := `
version: '3'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`
	path := s.createTestComposeFile(validContent)
	project, err := LoadComposeFile(path)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), project)
	assert.Equal(s.T(), 1, len(project.Services))
	assert.Equal(s.T(), "web", project.Services[0].Name)
	assert.Equal(s.T(), "nginx:latest", project.Services[0].Image)

	// Test non-existent file
	_, err = LoadComposeFile("nonexistent.yml")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "compose file not found")

	// Test invalid compose file
	invalidContent := `invalid: yaml: content`
	path = s.createTestComposeFile(invalidContent)
	_, err = LoadComposeFile(path)
	assert.Error(s.T(), err)
}

func (s *ComposeTestSuite) TestApplyPortMappings() {
	project := &types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "web",
				Image: "nginx:latest",
				Ports: []types.ServicePortConfig{
					{
						Mode:      "host",
						Target:    80,
						Published: "80",
					},
				},
			},
		},
	}

	// Test updating existing port mapping
	mappings := []PortMapping{
		{
			Service:   "web",
			Host:      "8080",
			Container: "80",
		},
	}
	err := ApplyPortMappings(project, mappings)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "8080", project.Services[0].Ports[0].Published)

	// Test non-existent service
	mappings = []PortMapping{
		{
			Service:   "nonexistent",
			Host:      "8080",
			Container: "80",
		},
	}
	err = ApplyPortMappings(project, mappings)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "service nonexistent not found")
}

func (s *ComposeTestSuite) TestGenerateYAML() {
	project := &types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "web",
				Image: "nginx:latest",
				Ports: []types.ServicePortConfig{
					{
						Mode:      "host",
						Target:    80,
						Published: "8080",
					},
				},
				Environment: map[string]*string{
					"DEBUG": strPtr("true"),
				},
			},
		},
	}

	yaml, err := GenerateYAML(project)
	require.NoError(s.T(), err)
	assert.Contains(s.T(), yaml, "image: nginx:latest")
	assert.Contains(s.T(), yaml, "ports:")
	assert.Contains(s.T(), yaml, "- 8080:80")
	assert.Contains(s.T(), yaml, "DEBUG: true")
}

func (s *ComposeTestSuite) TestRunComposeOperation() {
	// Create a test compose file
	content := `
version: '3'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`
	path := s.createTestComposeFile(content)

	// Test successful operation
	mappings := []PortMapping{
		{
			Service:   "web",
			Host:      "8080",
			Container: "80",
		},
	}
	err := RunComposeOperation(path, mappings, []string{"ps"})
	require.NoError(s.T(), err)

	// Test invalid command
	err = RunComposeOperation(path, mappings, []string{"invalid"})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid command arguments")
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

func TestComposeSuite(t *testing.T) {
	suite.Run(t, new(ComposeTestSuite))
}
