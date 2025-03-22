package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfigStructs(t *testing.T) {
	// Create a sample configuration
	sampleYAML := `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    context: ./context
    ports:
      - service: web
        host: "8080"
        container: "80"
    env:
      DEBUG: "true"
      API_KEY: "secret"
    env_file: .env
    ingress:
      enabled: true
      hostname: test.example.com
      paths:
        - /api
        - /web
      ssl:
        provider: letsencrypt
    remote:
      repo: git@github.com:user/repo.git
      branch: main
`

	var config Config
	err := yaml.Unmarshal([]byte(sampleYAML), &config)
	assert.NoError(t, err)
	assert.Len(t, config.Projects, 1)

	project := config.Projects[0]
	assert.Equal(t, "test-project", project.Name)
	assert.Equal(t, "/path/to/project", project.Path)
	assert.Equal(t, "docker-compose.yml", project.ComposeFile)
	assert.Equal(t, "./context", project.Context)
	assert.Len(t, project.Ports, 1)
	assert.Equal(t, "web", project.Ports[0].Service)
	assert.Equal(t, "8080", project.Ports[0].Host)
	assert.Equal(t, "80", project.Ports[0].Container)
	assert.Equal(t, "true", project.Env["DEBUG"])
	assert.Equal(t, "secret", project.Env["API_KEY"])
	assert.Equal(t, ".env", project.EnvFile)
	assert.True(t, project.Ingress.Enabled)
	assert.Equal(t, "test.example.com", project.Ingress.Hostname)
	assert.Equal(t, "letsencrypt", project.Ingress.SSL.Provider)
	assert.Equal(t, "git@github.com:user/repo.git", project.Remote.Repo)
	assert.Equal(t, "main", project.Remote.Branch)
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary test config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write test config to file
	testConfig := `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      DEBUG: "true"
`
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	assert.NoError(t, err)

	// Test successful config loading
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Projects, 1)
	assert.Equal(t, "test-project", config.Projects[0].Name)

	// Test non-existent file
	_, err = LoadConfig("/non/existent/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")

	// Test invalid YAML
	invalidConfig := `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      DEBUG: "true"
      invalid: yaml: : : :
`
	invalidConfigPath := filepath.Join(tmpDir, "invalid.yaml")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644)
	assert.NoError(t, err)

	_, err = LoadConfig(invalidConfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML config")

	// Test empty projects list
	emptyConfig := `
projects: []
`
	emptyConfigPath := filepath.Join(tmpDir, "empty.yaml")
	err = os.WriteFile(emptyConfigPath, []byte(emptyConfig), 0644)
	assert.NoError(t, err)

	_, err = LoadConfig(emptyConfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must contain at least one project")
}
