package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// BaseSuite provides common functionality for all test suites
type BaseSuite struct {
	suite.Suite
	TempDir string
}

// SetupSuite creates a temporary directory for test files
func (s *BaseSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "quay-test-*")
	require.NoError(s.T(), err)
	s.TempDir = tempDir
}

// TearDownSuite cleans up the temporary directory
func (s *BaseSuite) TearDownSuite() {
	if s.TempDir != "" {
		_ = os.RemoveAll(s.TempDir)
	}
}

// CreateTempFile creates a temporary file with the given content
func (s *BaseSuite) CreateTempFile(name, content string) string {
	path := filepath.Join(s.TempDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(s.T(), err)
	return path
}

// RunSuite runs a test suite with proper cleanup
func RunSuite(t *testing.T, s suite.TestingSuite) {
	suite.Run(t, s)
}
