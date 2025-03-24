package ingress

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	nginxContainerName = "nginx"
)

// NginxManager handles Nginx process management and configuration in a Docker container
type NginxManager struct {
	ConfigDir  string // Directory containing Nginx configuration files
	ConfigFile string // Name of the Nginx configuration file
}

// NewNginxManager creates a new NginxManager instance
func NewNginxManager(configDir, configFile string) *NginxManager {
	return &NginxManager{
		ConfigDir:  configDir,
		ConfigFile: configFile,
	}
}

// WriteConfig writes the provided Nginx configuration to a file
func (m *NginxManager) WriteConfig(config string) error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(m.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write the configuration file
	configPath := filepath.Join(m.ConfigDir, m.ConfigFile)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write Nginx configuration: %w", err)
	}

	return nil
}

// ReloadNginx reloads the Nginx configuration in the Docker container
func (m *NginxManager) ReloadNginx() error {
	// Reload Nginx in the Docker container
	cmd := exec.Command("docker", "exec", nginxContainerName, "nginx", "-s", "reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload Nginx in container: %s: %w", string(output), err)
	}

	return nil
}

// RestartNginx restarts the Nginx container
func (m *NginxManager) RestartNginx() error {
	// Restart the Nginx container
	cmd := exec.Command("docker", "restart", nginxContainerName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart Nginx container: %s: %w", string(output), err)
	}

	return nil
}

// CheckNginxStatus checks if the Nginx container is running
func (m *NginxManager) CheckNginxStatus() error {
	// Check container status
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", nginxContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check Nginx container status: %w", err)
	}
	if string(output) != "true\n" {
		return fmt.Errorf("nginx container is not running")
	}

	return nil
}
