package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	// DirName is the name of the cache directory
	DirName = ".quay/cache"
	// SSLSubDir is the subdirectory for SSL certificates
	SSLSubDir = "ssl"
	// NginxSubDir is the subdirectory for Nginx configurations
	NginxSubDir = "nginx"
	// ComposeSubDir is the subdirectory for compose files
	ComposeSubDir = "compose"
)

// GetCacheDir returns the path to the cache directory for the given project.
// It creates the directory if it doesn't exist.
func GetCacheDir(projectDir string) (string, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "GetCacheDir",
		"project":  projectDir,
	})

	cacheDir := filepath.Join(projectDir, DirName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logger.WithError(err).Error("Failed to create cache directory")
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	logger.Debug("Successfully created/accessed cache directory")
	return cacheDir, nil
}

// GetSubDir returns the path to a subdirectory within the cache directory.
// It creates the subdirectory if it doesn't exist.
func GetSubDir(projectDir, subDir string) (string, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "GetSubDir",
		"project":  projectDir,
		"subdir":   subDir,
	})

	cacheDir, err := GetCacheDir(projectDir)
	if err != nil {
		return "", err
	}

	subDirPath := filepath.Join(cacheDir, subDir)
	if err := os.MkdirAll(subDirPath, 0755); err != nil {
		logger.WithError(err).Error("Failed to create subdirectory")
		return "", fmt.Errorf("failed to create subdirectory %s: %w", subDir, err)
	}

	logger.Debug("Successfully created/accessed subdirectory")
	return subDirPath, nil
}

// GetSSLDir returns the path to the SSL certificates directory.
func GetSSLDir(projectDir string) (string, error) {
	return GetSubDir(projectDir, SSLSubDir)
}

// GetNginxDir returns the path to the Nginx configurations directory.
func GetNginxDir(projectDir string) (string, error) {
	return GetSubDir(projectDir, NginxSubDir)
}

// GetComposeDir returns the path to the compose files directory.
func GetComposeDir(projectDir string) (string, error) {
	return GetSubDir(projectDir, ComposeSubDir)
}
