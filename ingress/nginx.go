package ingress

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// NginxManager handles Nginx process management and configuration
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

// ReloadNginx reloads the Nginx configuration
func (m *NginxManager) ReloadNginx() error {
	// Skip actual Nginx operations in test mode
	if os.Getenv("QUAY_TEST_MODE") == "true" {
		return nil
	}

	// Check if Nginx is running in a Docker container
	containerName := os.Getenv("QUAY_NGINX_CONTAINER")
	if containerName != "" {
		// Reload Nginx in the Docker container
		cmd := exec.Command("docker", "exec", containerName, "nginx", "-s", "reload")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to reload Nginx in container: %s: %w", string(output), err)
		}
		return nil
	}

	// Reload local Nginx process
	cmd := exec.Command("nginx", "-s", "reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload Nginx: %s: %w", string(output), err)
	}

	return nil
}

// RestartNginx restarts the Nginx process
func (m *NginxManager) RestartNginx() error {
	// Skip actual Nginx operations in test mode
	if os.Getenv("QUAY_TEST_MODE") == "true" {
		return nil
	}

	// Check if Nginx is running in a Docker container
	containerName := os.Getenv("QUAY_NGINX_CONTAINER")
	if containerName != "" {
		// Restart the Nginx container
		cmd := exec.Command("docker", "restart", containerName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to restart Nginx container: %s: %w", string(output), err)
		}
		return nil
	}

	// Restart local Nginx process
	cmd := exec.Command("nginx", "-s", "stop")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop Nginx: %s: %w", string(output), err)
	}

	cmd = exec.Command("nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start Nginx: %s: %w", string(output), err)
	}

	return nil
}

// CheckNginxStatus checks if Nginx is running and accessible
func (m *NginxManager) CheckNginxStatus() error {
	// Skip actual Nginx operations in test mode
	if os.Getenv("QUAY_TEST_MODE") == "true" {
		return fmt.Errorf("Nginx is not running") // Simulate Nginx not running in test mode
	}

	// Check if Nginx is running in a Docker container
	containerName := os.Getenv("QUAY_NGINX_CONTAINER")
	if containerName != "" {
		// Check container status
		cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to check Nginx container status: %w", err)
		}
		if string(output) != "true\n" {
			return fmt.Errorf("Nginx container is not running")
		}
		return nil
	}

	// Check local Nginx process
	cmd := exec.Command("nginx", "-t")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Nginx configuration test failed: %s: %w", string(output), err)
	}

	return nil
}
