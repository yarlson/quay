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
	_ = os.RemoveAll(s.tempDir)
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
name: test-project
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
name: test-project
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

func (s *ComposeTestSuite) TestValidation() {
	tests := []struct {
		name        string
		project     *types.Project
		expectError string
	}{
		{
			name: "valid project",
			project: &types.Project{
				Name: "test-project",
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
			},
			expectError: "",
		},
		{
			name: "missing project name",
			project: &types.Project{
				Services: []types.ServiceConfig{
					{
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			expectError: "project name is required",
		},
		{
			name: "no services",
			project: &types.Project{
				Name:     "test-project",
				Services: []types.ServiceConfig{},
			},
			expectError: "project must contain at least one service",
		},
		{
			name: "missing service name",
			project: &types.Project{
				Name: "test-project",
				Services: []types.ServiceConfig{
					{
						Image: "nginx:latest",
					},
				},
			},
			expectError: "service name is required",
		},
		{
			name: "missing service image",
			project: &types.Project{
				Name: "test-project",
				Services: []types.ServiceConfig{
					{
						Name: "web",
					},
				},
			},
			expectError: "image is required for service web",
		},
		{
			name: "invalid port mapping",
			project: &types.Project{
				Name: "test-project",
				Services: []types.ServiceConfig{
					{
						Name:  "web",
						Image: "nginx:latest",
						Ports: []types.ServicePortConfig{
							{
								Mode: "host",
							},
						},
					},
				},
			},
			expectError: "container port is required for service web port mapping 0",
		},
		{
			name: "invalid volume mapping",
			project: &types.Project{
				Name: "test-project",
				Services: []types.ServiceConfig{
					{
						Name:  "web",
						Image: "nginx:latest",
						Volumes: []types.ServiceVolumeConfig{
							{
								Source: "/data",
							},
						},
					},
				},
			},
			expectError: "volume target is required for service web volume 0",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateProject(tt.project)
			if tt.expectError != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ComposeTestSuite) TestEnvVarSubstitution() {
	// Set up test environment variables
	err := os.Setenv("TEST_API_KEY", "secret123")
	s.Require().NoError(err, "Failed to set TEST_API_KEY")
	err = os.Setenv("TEST_DEBUG", "true")
	s.Require().NoError(err, "Failed to set TEST_DEBUG")
	defer func() {
		_ = os.Unsetenv("TEST_API_KEY")
		_ = os.Unsetenv("TEST_DEBUG")
	}()

	// Verify environment variables are set
	val, ok := os.LookupEnv("TEST_API_KEY")
	s.True(ok, "TEST_API_KEY should be set")
	s.Equal("secret123", val, "TEST_API_KEY should be set to secret123")
	val, ok = os.LookupEnv("TEST_DEBUG")
	s.True(ok, "TEST_DEBUG should be set")
	s.Equal("true", val, "TEST_DEBUG should be set to true")

	// Create a test compose file with environment variables
	content := `
name: test-project
version: '3'
services:
  web:
    image: nginx:latest
    environment:
      API_KEY: ${TEST_API_KEY}
      DEBUG: ${TEST_DEBUG:-false}
      UNSET_VAR: ${NON_EXISTENT_VAR:-default_value}
      COMPLEX_VAR: ${TEST_API_KEY}:${TEST_DEBUG}
`
	path := s.createTestComposeFile(content)

	// Load the compose file
	project, err := LoadComposeFile(path)
	s.Require().NoError(err)
	s.NotNil(project)

	// Check environment variables
	webService := project.Services[0]
	s.Equal("secret123", *webService.Environment["API_KEY"])
	s.Equal("true", *webService.Environment["DEBUG"])
	s.Equal("default_value", *webService.Environment["UNSET_VAR"])
	s.Equal("secret123:true", *webService.Environment["COMPLEX_VAR"])

	// Test invalid environment variable syntax
	invalidContent := `
name: test-project
version: '3'
services:
  web:
    image: nginx:latest
    environment:
      INVALID: ${TEST_API_KEY
`
	path = s.createTestComposeFile(invalidContent)
	_, err = LoadComposeFile(path)
	s.Error(err)
	s.Contains(err.Error(), "invalid interpolation format")
}

func TestComposeSuite(t *testing.T) {
	suite.Run(t, new(ComposeTestSuite))
}
