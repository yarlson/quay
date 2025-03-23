package ingress

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateSSLCerts generates SSL certificates for the given Config using mkcert.
// It updates the Config with the paths to the generated certificate and key files.
func GenerateSSLCerts(config *Config) error {
	if !config.SSLEnabled {
		return nil
	}

	// Get SSL directory from environment variable or use default
	sslDir := os.Getenv("QUAY_SSL_DIR")
	if sslDir == "" {
		sslDir = "/etc/nginx/ssl"
	}

	// Create SSL directory if it doesn't exist
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return fmt.Errorf("failed to create SSL directory: %w", err)
	}

	// Generate paths for certificate and key files
	certPath := filepath.Join(sslDir, config.Hostname+".crt")
	keyPath := filepath.Join(sslDir, config.Hostname+".key")

	// Skip actual certificate generation in test mode
	if os.Getenv("QUAY_TEST_MODE") == "true" {
		// In test mode, don't set paths at all
		return nil
	}

	// Check if mkcert is available
	if _, err := exec.LookPath("mkcert"); err != nil {
		return fmt.Errorf("mkcert not found: %w", err)
	}

	// Generate SSL certificate
	cmd := exec.Command("mkcert", "-install", "-cert-file", certPath, "-key-file", keyPath, config.Hostname)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate SSL certificate: %s: %w", string(output), err)
	}

	// Update the Config with the certificate paths
	config.SSLCertPath = certPath
	config.SSLKeyPath = keyPath

	return nil
}
