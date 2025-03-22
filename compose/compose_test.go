package compose

import (
	"os"
	"path/filepath"
	"testing"

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
