package lifecycle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yarlson/quay/compose"
	"github.com/yarlson/quay/config"
	"github.com/yarlson/quay/ingress"
)

// Manager handles project lifecycle operations
type Manager struct {
	configFile string
	project    string
	branch     string
}

// NewManager creates a new lifecycle manager
func NewManager(configFile, project, branch string) *Manager {
	return &Manager{
		configFile: configFile,
		project:    project,
		branch:     branch,
	}
}

// ProcessProjects processes projects based on the command name
func (m *Manager) ProcessProjects(cmdName string) error {
	// Check if config file is provided
	if m.configFile == "" {
		return fmt.Errorf("config file is required")
	}

	// Load the configuration file
	cfg, err := config.LoadConfig(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Filter projects if project name is specified
	var projects []config.Project
	if m.project != "" {
		found := false
		for _, p := range cfg.Projects {
			if p.Name == m.project {
				projects = append(projects, p)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("project %s not found in configuration", m.project)
		}
	} else {
		projects = cfg.Projects
	}

	// Process each project
	for _, project := range projects {
		// Override branch if specified
		if m.branch != "" && project.Remote != nil {
			project.Remote.Branch = m.branch
		}

		// Handle remote project provisioning
		if project.Remote != nil {
			if err := m.handleRemoteProject(&project); err != nil {
				return fmt.Errorf("failed to handle remote project %s: %w", project.Name, err)
			}
		}

		// Execute the requested command
		var err error
		switch cmdName {
		case "start":
			err = m.startProject(&project)
		case "stop":
			err = m.stopProject(&project)
		case "reload":
			err = m.reloadProject(&project)
		case "status":
			err = m.statusProject(&project)
		case "cleanup":
			err = m.cleanupProject(&project)
		default:
			return fmt.Errorf("invalid command: %s", cmdName)
		}

		if err != nil {
			return fmt.Errorf("failed to %s project %s: %w", cmdName, project.Name, err)
		}
	}

	return nil
}

// handleRemoteProject handles remote project provisioning
func (m *Manager) handleRemoteProject(project *config.Project) error {
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
func (m *Manager) startProject(project *config.Project) error {
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
func (m *Manager) stopProject(project *config.Project) error {
	return compose.RunComposeOperation(project.ComposeFile, nil, []string{"down"})
}

// reloadProject reloads a project
func (m *Manager) reloadProject(project *config.Project) error {
	// First stop the project
	if err := m.stopProject(project); err != nil {
		return err
	}

	// Then start it again
	return m.startProject(project)
}

// statusProject gets the status of a project
func (m *Manager) statusProject(project *config.Project) error {
	return compose.RunComposeOperation(project.ComposeFile, nil, []string{"ps"})
}

// cleanupProject cleans up a project
func (m *Manager) cleanupProject(project *config.Project) error {
	// Stop the project
	if err := m.stopProject(project); err != nil {
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
