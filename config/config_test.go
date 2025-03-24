package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	tmpDir string
}

func (s *ConfigTestSuite) SetupTest() {
	// Create a temporary directory for test files
	s.tmpDir = s.T().TempDir()

	// Save original logger and create a new one for tests
	testLogger := logrus.New()
	testLogger.SetFormatter(&logrus.JSONFormatter{})
	testLogger.SetLevel(logrus.DebugLevel)
}

func (s *ConfigTestSuite) createTestFile(name string, content string) string {
	path := filepath.Join(s.tmpDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	s.Require().NoError(err)
	return path
}

func (s *ConfigTestSuite) TestConfigStructs() {
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
	s.Require().NoError(err)
	s.Len(config.Projects, 1)

	project := config.Projects[0]
	s.Equal("test-project", project.Name)
	s.Equal("/path/to/project", project.Path)
	s.Equal("docker-compose.yml", project.ComposeFile)
	s.Equal("./context", project.Context)
	s.Len(project.Ports, 1)
	s.Equal("web", project.Ports[0].Service)
	s.Equal("8080", project.Ports[0].Host)
	s.Equal("80", project.Ports[0].Container)
	s.Equal("true", project.Env["DEBUG"])
	s.Equal("secret", project.Env["API_KEY"])
	s.Equal(".env", project.EnvFile)
	s.True(project.Ingress.Enabled)
	s.Equal("test.example.com", project.Ingress.Hostname)
	s.Equal("letsencrypt", project.Ingress.SSL.Provider)
	s.Equal("git@github.com:user/repo.git", project.Remote.Repo)
	s.Equal("main", project.Remote.Branch)
}

func (s *ConfigTestSuite) TestLoadConfig() {
	// Test successful config loading
	configPath := s.createTestFile("config.yaml", `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      DEBUG: "true"
`)

	config, err := LoadConfig(configPath)
	s.Require().NoError(err)
	s.NotNil(config)
	s.Len(config.Projects, 1)
	s.Equal("test-project", config.Projects[0].Name)

	// Test non-existent file
	_, err = LoadConfig("/non/existent/file.yaml")
	s.Error(err)
	s.Contains(err.Error(), "failed to read config file")

	// Test invalid YAML
	invalidConfigPath := s.createTestFile("invalid.yaml", `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      DEBUG: "true"
      invalid: yaml: : : :
`)

	_, err = LoadConfig(invalidConfigPath)
	s.Error(err)
	s.Contains(err.Error(), "failed to parse YAML config")

	// Test empty projects list
	emptyConfigPath := s.createTestFile("empty.yaml", `
projects: []
`)

	_, err = LoadConfig(emptyConfigPath)
	s.Error(err)
	s.Contains(err.Error(), "must contain at least one project")
}

func (s *ConfigTestSuite) TestEnvVarSubstitution() {
	// Set up test environment variables
	_ = os.Setenv("TEST_API_KEY", "secret123")
	_ = os.Setenv("TEST_DEBUG", "true")
	defer func() {
		_ = os.Unsetenv("TEST_API_KEY")
		_ = os.Unsetenv("TEST_DEBUG")
	}()

	configPath := s.createTestFile("config.yaml", `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      API_KEY: "${TEST_API_KEY}"
      DEBUG: "${TEST_DEBUG:-false}"
      UNSET_VAR: "${NON_EXISTENT_VAR:-default_value}"
      COMPLEX_VAR: "${TEST_API_KEY}:${TEST_DEBUG}"
`)

	config, err := LoadConfig(configPath)
	s.Require().NoError(err)
	s.NotNil(config)
	s.Len(config.Projects, 1)

	project := config.Projects[0]
	s.Equal("secret123", project.Env["API_KEY"], "Should substitute existing env var")
	s.Equal("true", project.Env["DEBUG"], "Should substitute existing env var")
	s.Equal("default_value", project.Env["UNSET_VAR"], "Should use default value for unset var")
	s.Equal("secret123:true", project.Env["COMPLEX_VAR"], "Should handle multiple substitutions")

	// Test invalid syntax
	invalidConfigPath := s.createTestFile("invalid_env.yaml", `
projects:
  - name: test-project
    path: /path/to/project
    compose_file: docker-compose.yml
    env:
      INVALID: "${TEST_API_KEY"
`)

	_, err = LoadConfig(invalidConfigPath)
	s.Error(err)
	s.Contains(err.Error(), "failed to process environment variables")
}

func (s *ConfigTestSuite) TestEnvFileLoading() {
	// Create a test .env file
	s.createTestFile(".env", `
# Test environment variables
API_KEY=secret123
DEBUG=true
# Commented out variable
#DISABLED=false
# Variable with spaces
COMPLEX_VAR=value with spaces
# Variable with quotes
QUOTED_VAR="quoted value"
`)

	configPath := s.createTestFile("config.yaml", `
projects:
  - name: test-project
    path: `+s.tmpDir+`
    compose_file: docker-compose.yml
    env_file: .env
    env:
      OVERRIDE_VAR: "overridden"
`)

	config, err := LoadConfig(configPath)
	s.Require().NoError(err)
	s.NotNil(config)
	s.Len(config.Projects, 1)

	project := config.Projects[0]
	s.Equal("secret123", project.Env["API_KEY"], "Should load variable from .env file")
	s.Equal("true", project.Env["DEBUG"], "Should load variable from .env file")
	s.Equal("value with spaces", project.Env["COMPLEX_VAR"], "Should handle spaces in values")
	s.Equal("quoted value", project.Env["QUOTED_VAR"], "Should handle quoted values")
	s.Equal("overridden", project.Env["OVERRIDE_VAR"], "Should not override existing variables")

	// Test non-existent .env file
	invalidConfigPath := s.createTestFile("invalid_config.yaml", `
projects:
  - name: test-project
    path: `+s.tmpDir+`
    compose_file: docker-compose.yml
    env_file: non-existent.env
`)

	_, err = LoadConfig(invalidConfigPath)
	s.Error(err)
	s.Contains(err.Error(), "failed to load .env file")

	// Test absolute path resolution
	absEnvPath := filepath.Join(s.tmpDir, "absolute.env")
	err = os.WriteFile(absEnvPath, []byte("TEST_VAR=absolute"), 0644)
	s.Require().NoError(err)

	absConfigPath := s.createTestFile("abs_config.yaml", `
projects:
  - name: test-project
    path: `+s.tmpDir+`
    compose_file: docker-compose.yml
    env_file: `+absEnvPath+`
`)

	config, err = LoadConfig(absConfigPath)
	s.Require().NoError(err)
	s.Equal("absolute", config.Projects[0].Env["TEST_VAR"], "Should handle absolute paths")
}

func (s *ConfigTestSuite) TestValidation() {
	tests := []struct {
		name        string
		config      string
		expectError string
	}{
		{
			name: "missing project name",
			config: `
projects:
  - path: /test/path
    compose_file: docker-compose.yml
`,
			expectError: "project name is required",
		},
		{
			name: "missing project path",
			config: `
projects:
  - name: test-project
    compose_file: docker-compose.yml
`,
			expectError: "project path is required",
		},
		{
			name: "missing compose file",
			config: `
projects:
  - name: test-project
    path: /test/path
`,
			expectError: "compose_file is required",
		},
		{
			name: "invalid ingress config - missing hostname",
			config: `
projects:
  - name: test-project
    path: /test/path
    compose_file: docker-compose.yml
    ingress:
      enabled: true
      paths:
        - /api
`,
			expectError: "hostname is required when ingress is enabled",
		},
		{
			name: "invalid ingress config - no paths",
			config: `
projects:
  - name: test-project
    path: /test/path
    compose_file: docker-compose.yml
    ingress:
      enabled: true
      hostname: test.example.com
`,
			expectError: "at least one path is required when ingress is enabled",
		},
		{
			name: "invalid remote config - missing repo",
			config: `
projects:
  - name: test-project
    path: /test/path
    compose_file: docker-compose.yml
    remote:
      branch: main
`,
			expectError: "repo is required when remote is configured",
		},
		{
			name: "valid config with default remote branch",
			config: `
projects:
  - name: test-project
    path: /test/path
    compose_file: docker-compose.yml
    remote:
      repo: git@github.com:user/repo.git
`,
			expectError: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			configPath := s.createTestFile("config.yaml", tt.config)
			config, err := LoadConfig(configPath)
			if tt.expectError != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectError)
			} else {
				s.NoError(err)
				s.NotNil(config)
				if config.Projects[0].Remote != nil {
					s.Equal("main", config.Projects[0].Remote.Branch)
				}
			}
		})
	}
}

func (s *ConfigTestSuite) TestPathProcessing() {
	// Create project directory and .env file
	projectDir := filepath.Join(s.tmpDir, "project")
	err := os.MkdirAll(projectDir, 0755)
	s.Require().NoError(err)

	envFile := filepath.Join(projectDir, ".env")
	err = os.WriteFile(envFile, []byte("TEST=value"), 0644)
	s.Require().NoError(err)

	// Create a test config file with relative paths
	configPath := s.createTestFile("config.yaml", `
projects:
  - name: test-project
    path: ./project
    compose_file: docker-compose.yml
    context: ./build
    env_file: .env
`)

	config, err := LoadConfig(configPath)
	s.Require().NoError(err)
	s.NotNil(config)

	project := config.Projects[0]
	configDir := filepath.Dir(configPath)

	// Check that paths are made absolute
	s.Equal(filepath.Join(configDir, "project"), project.Path)
	s.Equal(filepath.Join(configDir, "project", "docker-compose.yml"), project.ComposeFile)
	s.Equal(filepath.Join(configDir, "project", "build"), project.Context)

	// Check that environment variables were loaded
	s.Equal("value", project.Env["TEST"])
}

func (s *ConfigTestSuite) TestEnvVarSubstitutionInPaths() {
	// Set up test environment variables
	_ = os.Setenv("TEST_PROJECT_PATH", "/custom/path")
	_ = os.Setenv("TEST_COMPOSE_FILE", "custom-compose.yml")
	_ = os.Setenv("TEST_HOSTNAME", "test.example.com")
	_ = os.Setenv("TEST_REPO", "git@github.com:user/repo.git")
	defer func() {
		_ = os.Unsetenv("TEST_PROJECT_PATH")
		_ = os.Unsetenv("TEST_COMPOSE_FILE")
		_ = os.Unsetenv("TEST_HOSTNAME")
		_ = os.Unsetenv("TEST_REPO")
	}()

	configPath := s.createTestFile("config.yaml", `
projects:
  - name: test-project
    path: "${TEST_PROJECT_PATH}"
    compose_file: "${TEST_COMPOSE_FILE}"
    ingress:
      enabled: true
      hostname: "${TEST_HOSTNAME}"
      paths:
        - /api
    remote:
      repo: "${TEST_REPO}"
      branch: "${NON_EXISTENT_BRANCH:-main}"
`)

	config, err := LoadConfig(configPath)
	s.Require().NoError(err)
	s.NotNil(config)

	project := config.Projects[0]
	s.Equal("/custom/path", project.Path)
	s.Equal("/custom/path/custom-compose.yml", project.ComposeFile)
	s.Equal("test.example.com", project.Ingress.Hostname)
	s.Equal("git@github.com:user/repo.git", project.Remote.Repo)
	s.Equal("main", project.Remote.Branch)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
