// Package test provides  integration tests for XMLUI Local Server.
//
// This package contains end-to-end tests that validate server functionality.
package test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-testutil"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

var (
	// testDataDir is the shared test data directory for all integration tests
	testDataDir string
	// bootstrapSQL contains the shared bootstrap SQL content
	bootstrapSQL []byte
)

// TestMain sets up the test environment and runs all integration tests.
func DataDir() string {
	return testDataDir
}
func BootstrapSQL() string {
	return string(bootstrapSQL)
}
func TestMain(m *testing.M) {
	// Setup test environment
	if err := setup(); err != nil {
		cliutil.Stderrf("Setup failed: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup test environment
	teardown()

	os.Exit(code)
}

// setup prepares the test environment before running tests.
func setup() error {
	var err error

	// This ensures the logger is set up before cfgldr package initialization
	logger := slog.New(testutil.NewBufferedLogHandler())
	localsvr.SetLogger(logger)

	// Setup common test data that all tests can use
	wd, _ := os.Getwd()
	testDataDir = filepath.Join(wd, "test-data")

	// Generate the  test config file
	configPath := filepath.Join(testDataDir, "api__test.json")
	if err = generateConfig(configPath); err != nil {
		return fmt.Errorf("failed to generate  config: %w", err)
	}

	// Pre-load the bootstrap SQL that all tests will use
	sqlFile := filepath.Join(testDataDir, "bootstrap.sql")
	bootstrapSQL, err = os.ReadFile(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", sqlFile, err)
	}

	return nil
}

// teardown cleans up the test environment after all tests have run.
func teardown() {
	// Cleanup code here if needed in the future
}
