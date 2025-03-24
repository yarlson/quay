package ingress

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/yarlson/quay/cache"
)

const (
	nginxContainerName = "nginx"
	nginxConfigFile    = "quay.conf"
)

// NginxManager handles Nginx process management and configuration in a Docker container
type NginxManager struct {
	ConfigDir  string // Directory containing Nginx configuration files
	ConfigFile string // Name of the Nginx configuration file
}

// NewNginxManager creates a new NginxManager instance
func NewNginxManager(projectDir string) (*NginxManager, error) {
	// Get Nginx directory from cache
	nginxDir, err := cache.GetNginxDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get Nginx directory: %w", err)
	}

	return &NginxManager{
		ConfigDir:  nginxDir,
		ConfigFile: nginxConfigFile,
	}, nil
}

// WriteConfig writes the provided Nginx configuration to a file
func (m *NginxManager) WriteConfig(config string) error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "WriteConfig",
	})

	// Write the configuration file
	configPath := filepath.Join(m.ConfigDir, m.ConfigFile)
	logger.WithField("configPath", configPath).Debug("Writing Nginx configuration file")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		logger.WithError(err).Error("Failed to write Nginx configuration file")
		return fmt.Errorf("failed to write Nginx configuration: %w", err)
	}

	logger.Info("Successfully wrote Nginx configuration")
	return nil
}

// ReloadNginx reloads the Nginx configuration in the Docker container
func (m *NginxManager) ReloadNginx() error {
	logger := logrus.WithFields(logrus.Fields{
		"function":  "ReloadNginx",
		"container": nginxContainerName,
	})

	logger.Debug("Reloading Nginx configuration in container")
	cmd := exec.Command("docker", "exec", nginxContainerName, "nginx", "-s", "reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.WithFields(logrus.Fields{
			"error":  err,
			"output": string(output),
		}).Error("Failed to reload Nginx in container")
		return fmt.Errorf("failed to reload Nginx in container: %s: %w", string(output), err)
	}

	logger.Info("Successfully reloaded Nginx configuration")
	return nil
}

// RestartNginx restarts the Nginx container
func (m *NginxManager) RestartNginx() error {
	logger := logrus.WithFields(logrus.Fields{
		"function":  "RestartNginx",
		"container": nginxContainerName,
	})

	logger.Debug("Restarting Nginx container")
	cmd := exec.Command("docker", "restart", nginxContainerName)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.WithFields(logrus.Fields{
			"error":  err,
			"output": string(output),
		}).Error("Failed to restart Nginx container")
		return fmt.Errorf("failed to restart Nginx container: %s: %w", string(output), err)
	}

	logger.Info("Successfully restarted Nginx container")
	return nil
}

// CheckNginxStatus checks if the Nginx container is running
func (m *NginxManager) CheckNginxStatus() error {
	logger := logrus.WithFields(logrus.Fields{
		"function":  "CheckNginxStatus",
		"container": nginxContainerName,
	})

	logger.Debug("Checking Nginx container status")
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", nginxContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.WithError(err).Error("Failed to check Nginx container status")
		return fmt.Errorf("failed to check Nginx container status: %w", err)
	}
	if string(output) != "true\n" {
		logger.Error("Nginx container is not running")
		return fmt.Errorf("nginx container is not running")
	}

	logger.Info("Nginx container is running")
	return nil
}
