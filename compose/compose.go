package compose

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
)

// LoadComposeFile loads and parses a Docker Compose file from the provided path.
// It returns a types.Project containing the parsed configuration and any error encountered.
func LoadComposeFile(path string) (*types.Project, error) {
	logger := logrus.WithField("compose_file", path)

	// Ensure the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("compose file not found at %s", path)
	}

	// Get the absolute path of the compose file
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the directory containing the compose file
	workingDir := filepath.Dir(absPath)

	// Load the compose file
	config, err := loader.Load(types.ConfigDetails{
		WorkingDir: workingDir,
		ConfigFiles: []types.ConfigFile{
			{
				Filename: absPath,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load compose file: %w", err)
	}

	logger.Debug("Successfully loaded compose file")
	return config, nil
}
