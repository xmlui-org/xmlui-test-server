package test

import (
	"os"
	"testing"

	testutil "github.com/mikeschinkel/go-testutil"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

const testDataDir = "./test-data"

// TestMain sets up the test environment and runs all integration tests.
func TestMain(m *testing.M) {

	logger := testutil.NewNullLogger()
	localsvr.SetLogger(logger) // TODO Change to a buffered logger

	// Run tests
	code := m.Run()

	// Cleanup code here if needed

	os.Exit(code)
}
