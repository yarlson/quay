package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type ConfigTestSuite struct {
	suite.Suite
	tmpDir     string
	logBuffer  *bytes.Buffer
	origLogger *logrus.Logger
}

func (s *ConfigTestSuite) SetupTest() {
	// Create a temporary directory for test files
	s.tmpDir = s.T().TempDir()

	// Save original logger and create a new one for tests
	s.origLogger = log
	s.logBuffer = new(bytes.Buffer)
	testLogger := logrus.New()
	testLogger.SetOutput(s.logBuffer)
	testLogger.SetFormatter(&logrus.JSONFormatter{})
	testLogger.SetLevel(logrus.DebugLevel)
	log = testLogger
}

func (s *ConfigTestSuite) TearDownTest() {
	// Restore original logger
	log = s.origLogger
}

func (s *ConfigTestSuite) createTestFile(name string, content string) string {
	path := filepath.Join(s.tmpDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	s.Require().NoError(err)
	return path
}

func (s *ConfigTestSuite) getLogEntries() []map[string]interface{} {
	var entries []map[string]interface{}
	for _, line := range bytes.Split(s.logBuffer.Bytes(), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]interface{}
		err := json.Unmarshal(line, &entry)
		s.Require().NoError(err)
		entries = append(entries, entry)
	}
	return entries
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
	os.Setenv("TEST_API_KEY", "secret123")
	os.Setenv("TEST_DEBUG", "true")
	defer func() {
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("TEST_DEBUG")
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

func (s *ConfigTestSuite) TestLogging() {
	configPath := s.createTestFile("config.yaml", `
projects:
  - name: test-project
    path: `+s.tmpDir+`
    compose_file: docker-compose.yml
    env_file: .env
    env:
      TEST_VAR: "${TEST_VAR:-default}"
`)

	s.createTestFile(".env", `
TEST_VAR=from_env
`)

	// Clear log buffer
	s.logBuffer.Reset()

	config, err := LoadConfig(configPath)
	s.Require().NoError(err)
	s.NotNil(config)

	// Get log entries
	entries := s.getLogEntries()
	s.Greater(len(entries), 0, "Should have log entries")

	// Check for specific log entries
	foundInfo := false
	foundDebug := false
	foundError := false

	for _, entry := range entries {
		level, ok := entry["level"].(string)
		s.True(ok, "Log entry should have a level")

		switch level {
		case "info":
			foundInfo = true
			msg, ok := entry["msg"].(string)
			s.True(ok)
			if msg == "Loading configuration file" {
				s.Contains(entry, "file", "Info log should have file field")
			}
		case "debug":
			foundDebug = true
			msg, ok := entry["msg"].(string)
			s.True(ok)
			if msg == "Substituting environment variables" {
				s.Contains(entry, "value", "Debug log should have value field")
			}
		case "error":
			foundError = true
		}
	}

	s.True(foundInfo, "Should have info level logs")
	s.True(foundDebug, "Should have debug level logs")
	s.False(foundError, "Should not have error level logs")

	// Test error logging
	log.SetLevel(logrus.ErrorLevel)
	s.logBuffer.Reset()

	// Try to load a non-existent file
	_, err = LoadConfig("/non/existent/file.yaml")
	s.Error(err)

	// Verify error log entry
	entries = s.getLogEntries()
	s.Require().NotEmpty(entries)
	errorEntry := entries[0]

	level, ok := errorEntry["level"].(string)
	s.True(ok)
	s.Equal("error", level)

	msg, ok := errorEntry["msg"].(string)
	s.True(ok)
	s.Contains(msg, "Failed to read config file")

	// Test log level from environment variable
	os.Setenv("QUAY_LOG_LEVEL", "debug")
	defer os.Unsetenv("QUAY_LOG_LEVEL")

	InitLogger()
	s.Equal(logrus.DebugLevel, log.GetLevel())
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
