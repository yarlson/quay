package lifecycle

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
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
	logger := logrus.WithFields(logrus.Fields{
		"command": cmdName,
		"config":  m.configFile,
	})

	logger.Debug("Starting project processing")

	// Check if config file is provided
	if m.configFile == "" {
		logger.Error("Config file is required")
		return fmt.Errorf("config file is required")
	}

	// Load the configuration file
	cfg, err := config.LoadConfig(m.configFile)
	if err != nil {
		logger.WithError(err).Error("Failed to load configuration")
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
			logger.WithField("project", m.project).Error("Project not found in configuration")
			return fmt.Errorf("project %s not found in configuration", m.project)
		}
	} else {
		projects = cfg.Projects
	}

	logger.WithField("project_count", len(projects)).Debug("Processing projects")

	// Process each project
	for _, project := range projects {
		projectLogger := logger.WithField("project", project.Name)
		projectLogger.Debug("Processing project")

		// Override branch if specified
		if m.branch != "" && project.Remote != nil {
			project.Remote.Branch = m.branch
		}

		// Handle remote project provisioning
		if project.Remote != nil {
			if err := m.handleRemoteProject(&project); err != nil {
				projectLogger.WithError(err).Error("Failed to handle remote project")
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
			projectLogger.WithField("command", cmdName).Error("Invalid command")
			return fmt.Errorf("invalid command: %s", cmdName)
		}

		if err != nil {
			projectLogger.WithError(err).Errorf("Failed to %s project", cmdName)
			return fmt.Errorf("failed to %s project %s: %w", cmdName, project.Name, err)
		}

		projectLogger.Info("Successfully processed project")
	}

	logger.Info("Finished processing all projects")
	return nil
}

// handleRemoteProject handles remote project provisioning
func (m *Manager) handleRemoteProject(project *config.Project) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": project.Name,
		"repo":    project.Remote.Repo,
		"branch":  project.Remote.Branch,
	})

	logger.Debug("Handling remote project")

	// Create project directory if it doesn't exist
	if err := os.MkdirAll(project.Path, 0755); err != nil {
		logger.WithError(err).Error("Failed to create project directory")
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Check if the directory is empty
	entries, err := os.ReadDir(project.Path)
	if err != nil {
		logger.WithError(err).Error("Failed to read project directory")
		return fmt.Errorf("failed to read project directory: %w", err)
	}

	// If directory is empty, clone the repository
	if len(entries) == 0 {
		logger.Debug("Cloning repository")
		cmd := exec.Command("git", "clone", "-b", project.Remote.Branch, project.Remote.Repo, project.Path)
		if output, err := cmd.CombinedOutput(); err != nil {
			logger.WithError(err).WithField("output", string(output)).Error("Failed to clone repository")
			return fmt.Errorf("failed to clone repository: %s: %w", string(output), err)
		}
		logger.Info("Successfully cloned repository")
	} else {
		logger.Debug("Pulling latest changes")
		cmd := exec.Command("git", "-C", project.Path, "pull", "origin", project.Remote.Branch)
		if output, err := cmd.CombinedOutput(); err != nil {
			logger.WithError(err).WithField("output", string(output)).Error("Failed to pull latest changes")
			return fmt.Errorf("failed to pull latest changes: %s: %w", string(output), err)
		}
		logger.Info("Successfully pulled latest changes")
	}

	return nil
}

// startProject starts a project
func (m *Manager) startProject(project *config.Project) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": project.Name,
		"compose": project.ComposeFile,
	})

	logger.Debug("Starting project")

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
	if err := compose.RunComposeOperation(project.ComposeFile, mappings, []string{"up", "-d"}); err != nil {
		logger.WithError(err).Error("Failed to start project")
		return err
	}

	logger.Info("Successfully started project")
	return nil
}

// stopProject stops a project
func (m *Manager) stopProject(project *config.Project) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": project.Name,
		"compose": project.ComposeFile,
	})

	logger.Debug("Stopping project")

	if err := compose.RunComposeOperation(project.ComposeFile, nil, []string{"down"}); err != nil {
		logger.WithError(err).Error("Failed to stop project")
		return err
	}

	logger.Info("Successfully stopped project")
	return nil
}

// reloadProject reloads a project
func (m *Manager) reloadProject(project *config.Project) error {
	logger := logrus.WithField("project", project.Name)
	logger.Debug("Reloading project")

	// First stop the project
	if err := m.stopProject(project); err != nil {
		return err
	}

	// Then start it again
	if err := m.startProject(project); err != nil {
		return err
	}

	logger.Info("Successfully reloaded project")
	return nil
}

// statusProject gets the status of a project
func (m *Manager) statusProject(project *config.Project) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": project.Name,
		"compose": project.ComposeFile,
	})

	logger.Debug("Getting project status")

	if err := compose.RunComposeOperation(project.ComposeFile, nil, []string{"ps"}); err != nil {
		logger.WithError(err).Error("Failed to get project status")
		return err
	}

	logger.Info("Successfully retrieved project status")
	return nil
}

// cleanupProject cleans up a project
func (m *Manager) cleanupProject(project *config.Project) error {
	logger := logrus.WithField("project", project.Name)
	logger.Debug("Cleaning up project")

	// Stop the project
	if err := m.stopProject(project); err != nil {
		return err
	}

	// Remove the project directory if it's a remote project
	if project.Remote != nil {
		if err := os.RemoveAll(project.Path); err != nil {
			logger.WithError(err).Error("Failed to remove project directory")
			return fmt.Errorf("failed to remove project directory: %w", err)
		}
		logger.Debug("Removed project directory")
	}

	// Clean up ingress configuration if enabled
	if project.Ingress != nil && project.Ingress.Enabled {
		nginxManager, err := ingress.NewNginxManager(project.Path)
		if err != nil {
			logger.WithError(err).Error("Failed to create Nginx manager")
			return fmt.Errorf("failed to create Nginx manager: %w", err)
		}
		if err := nginxManager.WriteConfig(""); err != nil {
			logger.WithError(err).Error("Failed to clean up ingress configuration")
			return fmt.Errorf("failed to clean up ingress configuration: %w", err)
		}
		logger.Debug("Cleaned up ingress configuration")
	}

	logger.Info("Successfully cleaned up project")
	return nil
}
