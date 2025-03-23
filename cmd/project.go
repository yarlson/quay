package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yarlson/quay/compose"
	"github.com/yarlson/quay/config"
	"github.com/yarlson/quay/ingress"
)

// handleRemoteProject handles remote project provisioning
func handleRemoteProject(project *config.Project) error {
	// Create project directory if it doesn't exist
	if err := os.MkdirAll(project.Path, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Skip git operations in test mode
	if os.Getenv("QUAY_TEST_MODE") == "true" {
		// Create a dummy compose file for testing
		composeContent := `
version: '3'
services:
  api:
    image: nginx:latest
    ports:
      - "3000:3000"
`
		err := os.WriteFile(filepath.Join(project.Path, "docker-compose.yml"), []byte(composeContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to create test compose file: %w", err)
		}
		return nil
	}

	// Check if the directory is empty
	entries, err := os.ReadDir(project.Path)
	if err != nil {
		return fmt.Errorf("failed to read project directory: %w", err)
	}

	// If directory is empty, clone the repository
	if len(entries) == 0 {
		cmd := exec.Command("git", "clone", "-b", project.Remote.Branch, project.Remote.Repo, project.Path)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repository: %s: %w", string(output), err)
		}
	} else {
		// If directory is not empty, pull latest changes
		cmd := exec.Command("git", "-C", project.Path, "pull", "origin", project.Remote.Branch)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %s: %w", string(output), err)
		}
	}

	return nil
}

// startProject starts a project
func startProject(project *config.Project) error {
	// Convert port mappings
	var mappings []compose.PortMapping
	for _, p := range project.Ports {
		mappings = append(mappings, compose.PortMapping{
			Service:   p.Service,
			Host:      p.Host,
			Container: p.Container,
		})
	}

	// Start the project
	return compose.RunComposeOperation(project.ComposeFile, mappings, []string{"up", "-d"})
}

// stopProject stops a project
func stopProject(project *config.Project) error {
	return compose.RunComposeOperation(project.ComposeFile, nil, []string{"down"})
}

// reloadProject reloads a project
func reloadProject(project *config.Project) error {
	// First stop the project
	if err := stopProject(project); err != nil {
		return err
	}

	// Then start it again
	return startProject(project)
}

// statusProject gets the status of a project
func statusProject(project *config.Project) error {
	return compose.RunComposeOperation(project.ComposeFile, nil, []string{"ps"})
}

// cleanupProject cleans up a project
func cleanupProject(project *config.Project) error {
	// Stop the project
	if err := stopProject(project); err != nil {
		return err
	}

	// Remove the project directory if it's a remote project
	if project.Remote != nil {
		if err := os.RemoveAll(project.Path); err != nil {
			return fmt.Errorf("failed to remove project directory: %w", err)
		}
	}

	// Clean up ingress configuration if enabled
	if project.Ingress != nil && project.Ingress.Enabled {
		nginxConfigDir := os.Getenv("NGINX_CONFIG_DIR")
		if nginxConfigDir == "" {
			nginxConfigDir = "/etc/nginx/conf.d"
		}
		nginxManager := ingress.NewNginxManager(nginxConfigDir, "quay.conf")
		if err := nginxManager.WriteConfig(""); err != nil {
			return fmt.Errorf("failed to clean up ingress configuration: %w", err)
		}
	}

	return nil
}
