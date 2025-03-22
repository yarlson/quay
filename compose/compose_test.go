package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadComposeFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a sample docker-compose.yml file
	composeContent := `
version: '3.8'
services:
  test-service:
    image: nginx:latest
    ports:
      - "8080:80"
`
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	err = os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name          string
		path          string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid compose file",
			path:        composePath,
			expectError: false,
		},
		{
			name:          "non-existent file",
			path:          filepath.Join(tmpDir, "nonexistent.yml"),
			expectError:   true,
			errorContains: "compose file not found",
		},
		{
			name:          "invalid compose file",
			path:          filepath.Join(tmpDir, "invalid.yml"),
			expectError:   true,
			errorContains: "failed to load compose file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the invalid file test, create an invalid YAML file
			if tt.name == "invalid compose file" {
				err := os.WriteFile(tt.path, []byte("invalid: yaml: content: -"), 0644)
				require.NoError(t, err)
			}

			project, err := LoadComposeFile(tt.path)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, project)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, project)
			assert.Len(t, project.Services, 1)
			assert.Equal(t, "test-service", project.Services[0].Name)
		})
	}
}

func TestApplyPortMappings(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a sample docker-compose.yml file with multiple services and ports
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  api:
    image: node:latest
    ports:
      - "3000:3000"
`
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	err = os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Load the compose file
	project, err := LoadComposeFile(composePath)
	require.NoError(t, err)
	require.NotNil(t, project)

	tests := []struct {
		name          string
		mappings      []PortMapping
		expectError   bool
		errorContains string
		validate      func(*testing.T, *types.Project)
	}{
		{
			name: "update existing port mapping",
			mappings: []PortMapping{
				{
					Service:   "web",
					Host:      "8081",
					Container: "80",
				},
			},
			expectError: false,
			validate: func(t *testing.T, p *types.Project) {
				webService := findService(t, p, "web")
				assert.Equal(t, "8081", webService.Ports[0].Published)
			},
		},
		{
			name: "add new port mapping",
			mappings: []PortMapping{
				{
					Service:   "web",
					Host:      "8443",
					Container: "443",
				},
			},
			expectError: false,
			validate: func(t *testing.T, p *types.Project) {
				webService := findService(t, p, "web")
				assert.Len(t, webService.Ports, 2)
				assert.Equal(t, "8443", webService.Ports[1].Published)
			},
		},
		{
			name: "update multiple services",
			mappings: []PortMapping{
				{
					Service:   "web",
					Host:      "8082",
					Container: "80",
				},
				{
					Service:   "api",
					Host:      "3001",
					Container: "3000",
				},
			},
			expectError: false,
			validate: func(t *testing.T, p *types.Project) {
				webService := findService(t, p, "web")
				apiService := findService(t, p, "api")
				assert.Equal(t, "8082", webService.Ports[0].Published)
				assert.Equal(t, "3001", apiService.Ports[0].Published)
			},
		},
		{
			name: "non-existent service",
			mappings: []PortMapping{
				{
					Service:   "non-existent",
					Host:      "8080",
					Container: "80",
				},
			},
			expectError:   true,
			errorContains: "service non-existent not found in compose file",
		},
		{
			name: "invalid container port",
			mappings: []PortMapping{
				{
					Service:   "web",
					Host:      "8080",
					Container: "invalid",
				},
			},
			expectError:   true,
			errorContains: "invalid container port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the project for each test
			projectCopy := *project
			projectCopy.Services = make([]types.ServiceConfig, len(project.Services))
			copy(projectCopy.Services, project.Services)

			err := ApplyPortMappings(&projectCopy, tt.mappings)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			assert.NoError(t, err)
			tt.validate(t, &projectCopy)
		})
	}
}

func TestGenerateYAML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a sample project with multiple services and configurations
	project := &types.Project{
		Name: "test-project",
		Services: []types.ServiceConfig{
			{
				Name:  "web",
				Image: "nginx:latest",
				Ports: []types.ServicePortConfig{
					{
						Mode:      "host",
						Published: "8080",
						Target:    80,
					},
				},
				Environment: map[string]*string{
					"DEBUG": strPtr("true"),
				},
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "bind",
						Source: "./web",
						Target: "/usr/share/nginx/html",
					},
				},
			},
			{
				Name:  "api",
				Image: "node:latest",
				Ports: []types.ServicePortConfig{
					{
						Mode:      "host",
						Published: "3000",
						Target:    3000,
					},
				},
				Environment: map[string]*string{
					"NODE_ENV": strPtr("development"),
				},
			},
		},
	}

	// Generate YAML
	yamlData, err := GenerateYAML(project)
	require.NoError(t, err)
	require.NotEmpty(t, yamlData)

	// Write the generated YAML to a temporary file
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	err = os.WriteFile(composePath, []byte(yamlData), 0644)
	require.NoError(t, err)

	// Verify the generated YAML contains expected content
	assert.Contains(t, yamlData, "services:")
	assert.Contains(t, yamlData, "web:")
	assert.Contains(t, yamlData, "api:")
	assert.Contains(t, yamlData, "image: nginx:latest")
	assert.Contains(t, yamlData, "image: node:latest")
	assert.Contains(t, yamlData, "ports:")
	assert.Contains(t, yamlData, "- 8080:80")
	assert.Contains(t, yamlData, "- 3000:3000")
	assert.Contains(t, yamlData, "environment:")
	assert.Contains(t, yamlData, "DEBUG: true")
	assert.Contains(t, yamlData, "NODE_ENV: development")
	assert.Contains(t, yamlData, "volumes:")
	assert.Contains(t, yamlData, "- ./web:/usr/share/nginx/html")

	// Verify the YAML is valid by loading it back
	projectCopy, err := LoadComposeFile(composePath)
	require.NoError(t, err)
	require.NotNil(t, projectCopy)

	// Verify the loaded project has the same number of services
	assert.Equal(t, len(project.Services), len(projectCopy.Services))

	// Create maps for easier comparison
	originalServices := make(map[string]types.ServiceConfig)
	for _, service := range project.Services {
		originalServices[service.Name] = service
	}

	loadedServices := make(map[string]types.ServiceConfig)
	for _, service := range projectCopy.Services {
		loadedServices[service.Name] = service
	}

	// Compare services
	for name, originalService := range originalServices {
		loadedService, ok := loadedServices[name]
		assert.True(t, ok, "Service %s not found in loaded project", name)
		if ok {
			assert.Equal(t, originalService.Image, loadedService.Image)
			assert.Equal(t, len(originalService.Ports), len(loadedService.Ports))
			assert.Equal(t, len(originalService.Environment), len(loadedService.Environment))
			assert.Equal(t, len(originalService.Volumes), len(loadedService.Volumes))
		}
	}
}

// Helper function to find a service by name in a project
func findService(t *testing.T, p *types.Project, name string) *types.ServiceConfig {
	for i := range p.Services {
		if p.Services[i].Name == name {
			return &p.Services[i]
		}
	}
	t.Fatalf("service %s not found in project", name)
	return nil
}

// Helper function to create a string pointer
func strPtr(s string) *string {
	return &s
}
