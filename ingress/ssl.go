package ingress

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/yarlson/quay/cache"
)

// GenerateSSLCerts generates SSL certificates for the given Config using mkcert.
// It updates the Config with the paths to the generated certificate and key files.
func GenerateSSLCerts(config *Config, projectDir string) error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "GenerateSSLCerts",
		"hostname": config.Hostname,
	})

	if !config.SSLEnabled {
		logger.Debug("SSL is not enabled, skipping certificate generation")
		return nil
	}

	// Get SSL directory from cache
	sslDir, err := cache.GetSSLDir(projectDir)
	if err != nil {
		logger.WithError(err).Error("Failed to get SSL directory")
		return fmt.Errorf("failed to get SSL directory: %w", err)
	}

	logger.Debugf("Using SSL directory: %s", sslDir)

	// Generate paths for certificate and key files
	certPath := filepath.Join(sslDir, config.Hostname+".crt")
	keyPath := filepath.Join(sslDir, config.Hostname+".key")
	logger.Debugf("Certificate will be generated at: %s", certPath)

	// Check if mkcert is available
	if _, err := exec.LookPath("mkcert"); err != nil {
		logger.WithError(err).Error("mkcert not found in PATH")
		return fmt.Errorf("mkcert not found: %w", err)
	}

	// Generate SSL certificate
	cmd := exec.Command("mkcert", "-install", "-cert-file", certPath, "-key-file", keyPath, config.Hostname)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.WithError(err).WithField("output", string(output)).Error("Failed to generate SSL certificate")
		return fmt.Errorf("failed to generate SSL certificate: %s: %w", string(output), err)
	}

	// Update the Config with the certificate paths
	config.SSLCertPath = certPath
	config.SSLKeyPath = keyPath
	logger.Info("Successfully generated SSL certificate")

	return nil
}
