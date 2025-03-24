package ingress

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// GenerateSSLCerts generates SSL certificates for the given Config using mkcert.
// It updates the Config with the paths to the generated certificate and key files.
func GenerateSSLCerts(config *Config) error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "GenerateSSLCerts",
		"hostname": config.Hostname,
	})

	if !config.SSLEnabled {
		logger.Debug("SSL is not enabled, skipping certificate generation")
		return nil
	}

	sslDir := "/etc/nginx/ssl"
	logger.Debugf("Using SSL directory: %s", sslDir)

	// Create SSL directory if it doesn't exist
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		logger.WithError(err).Error("Failed to create SSL directory")
		return fmt.Errorf("failed to create SSL directory: %w", err)
	}

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
