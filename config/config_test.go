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

func TestEnvVarSubstitution(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_API_KEY", "secret123")
	os.Setenv("TEST_DEBUG", "true")
	defer func() {
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("TEST_DEBUG")
	}()

	// Create a temporary test config file with environment variables
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	testConfig := `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      API_KEY: "${TEST_API_KEY}"
      DEBUG: "${TEST_DEBUG:-false}"
      UNSET_VAR: "${NON_EXISTENT_VAR:-default_value}"
      COMPLEX_VAR: "${TEST_API_KEY}:${TEST_DEBUG}"
`
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	assert.NoError(t, err)

	// Test environment variable substitution
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Projects, 1)

	project := config.Projects[0]
	assert.Equal(t, "secret123", project.Env["API_KEY"], "Should substitute existing env var")
	assert.Equal(t, "true", project.Env["DEBUG"], "Should substitute existing env var")
	assert.Equal(t, "default_value", project.Env["UNSET_VAR"], "Should use default value for unset var")
	assert.Equal(t, "secret123:true", project.Env["COMPLEX_VAR"], "Should handle multiple substitutions")

	// Test invalid syntax
	invalidConfig := `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      INVALID: "${TEST_API_KEY"
`
	invalidConfigPath := filepath.Join(tmpDir, "invalid_env.yaml")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644)
	assert.NoError(t, err)

	_, err = LoadConfig(invalidConfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process environment variables")
}

func TestEnvFileLoading(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()

	// Create a test .env file
	envContent := `
# Test environment variables
API_KEY=secret123
DEBUG=true
# Commented out variable
#DISABLED=false
# Variable with spaces
COMPLEX_VAR=value with spaces
# Variable with quotes
QUOTED_VAR="quoted value"
`
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte(envContent), 0644)
	assert.NoError(t, err)

	// Create a test config file
	configContent := `
projects:
  - name: test-project
    path: ` + tmpDir + `
    compose_file: docker-compose.yml
    env_file: .env
    env:
      OVERRIDE_VAR: "overridden"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test .env file loading
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Projects, 1)

	project := config.Projects[0]
	assert.Equal(t, "secret123", project.Env["API_KEY"], "Should load variable from .env file")
	assert.Equal(t, "true", project.Env["DEBUG"], "Should load variable from .env file")
	assert.Equal(t, "value with spaces", project.Env["COMPLEX_VAR"], "Should handle spaces in values")
	assert.Equal(t, "quoted value", project.Env["QUOTED_VAR"], "Should handle quoted values")
	assert.Equal(t, "overridden", project.Env["OVERRIDE_VAR"], "Should not override existing variables")

	// Test non-existent .env file
	invalidConfig := `
projects:
  - name: test-project
    path: ` + tmpDir + `
    compose_file: docker-compose.yml
    env_file: non-existent.env
`
	invalidConfigPath := filepath.Join(tmpDir, "invalid_config.yaml")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644)
	assert.NoError(t, err)

	_, err = LoadConfig(invalidConfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load .env file")

	// Test absolute path resolution
	absEnvPath := filepath.Join(tmpDir, "absolute.env")
	err = os.WriteFile(absEnvPath, []byte("TEST_VAR=absolute"), 0644)
	assert.NoError(t, err)

	absConfig := `
projects:
  - name: test-project
    path: ` + tmpDir + `
    compose_file: docker-compose.yml
    env_file: ` + absEnvPath + `
`
	absConfigPath := filepath.Join(tmpDir, "abs_config.yaml")
	err = os.WriteFile(absConfigPath, []byte(absConfig), 0644)
	assert.NoError(t, err)

	config, err = LoadConfig(absConfigPath)
	assert.NoError(t, err)
	assert.Equal(t, "absolute", config.Projects[0].Env["TEST_VAR"], "Should handle absolute paths")
}
