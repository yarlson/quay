package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CacheTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *CacheTestSuite) SetupTest() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "cache-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
}

func (s *CacheTestSuite) TearDownTest() {
	// Clean up the temporary directory
	_ = os.RemoveAll(s.tempDir)
}

func (s *CacheTestSuite) TestGetCacheDir() {
	// Test creating cache directory
	cacheDir, err := GetCacheDir(s.tempDir)
	s.NoError(err)
	s.NotEmpty(cacheDir)
	s.True(filepath.IsAbs(cacheDir))
	s.Equal(filepath.Join(s.tempDir, DirName), cacheDir)

	// Verify directory exists
	_, err = os.Stat(cacheDir)
	s.NoError(err)

	// Test creating cache directory again (should not error)
	cacheDir2, err := GetCacheDir(s.tempDir)
	s.NoError(err)
	s.Equal(cacheDir, cacheDir2)
}

func (s *CacheTestSuite) TestGetSubDir() {
	// Test creating SSL subdirectory
	sslDir, err := GetSSLDir(s.tempDir)
	s.NoError(err)
	s.NotEmpty(sslDir)
	s.True(filepath.IsAbs(sslDir))
	s.Equal(filepath.Join(s.tempDir, DirName, SSLSubDir), sslDir)

	// Verify directory exists
	_, err = os.Stat(sslDir)
	s.NoError(err)

	// Test creating Nginx subdirectory
	nginxDir, err := GetNginxDir(s.tempDir)
	s.NoError(err)
	s.NotEmpty(nginxDir)
	s.True(filepath.IsAbs(nginxDir))
	s.Equal(filepath.Join(s.tempDir, DirName, NginxSubDir), nginxDir)

	// Verify directory exists
	_, err = os.Stat(nginxDir)
	s.NoError(err)

	// Test creating Compose subdirectory
	composeDir, err := GetComposeDir(s.tempDir)
	s.NoError(err)
	s.NotEmpty(composeDir)
	s.True(filepath.IsAbs(composeDir))
	s.Equal(filepath.Join(s.tempDir, DirName, ComposeSubDir), composeDir)

	// Verify directory exists
	_, err = os.Stat(composeDir)
	s.NoError(err)
}

func (s *CacheTestSuite) TestGetSubDirWithInvalidProjectDir() {
	// Test with non-existent project directory
	_, err := GetSubDir("/nonexistent/dir", SSLSubDir)
	s.Error(err)
	s.Contains(err.Error(), "failed to create cache directory")
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
