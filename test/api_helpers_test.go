package test

// api_helpers_test.go does not contain tests but is named with an _test.go
// suffix so Goland will stop flagging BootstrapSQL() and DataDir() as missing
// for some godforsaken reason.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cfgstore/cstest"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/go-fsfix"
	"github.com/mikeschinkel/go-rfc9457"
	"github.com/mikeschinkel/go-testutil"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apiresp"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

const (
	TestUsername = "gracehopper"
)

// testRequest represents a single HTTP request test case
type testRequest struct {
	name            string
	method          string
	path            string
	body            string
	expectedStatus  int
	expectedFields  []string          // Fields that should exist in response (for success cases)
	shouldContain   []string          // Content that should be in response (legacy, prefer expectedRFC9457)
	shouldNotHave   []string          // Content that should NOT be in response
	expectedRFC9457 *rfc9457.Response // Expected RFC 9457 error response (for error cases)
}

// testEnvironment holds the test environment setup for a single test case
type testEnvironment struct {
	rootFixture        *fsfix.RootFixture
	appInfo            appinfo.AppInfo
	dbPath             dt.Filepath
	bootstrapFile      *fsfix.FileFixture
	configFile         *fsfix.FileFixture
	configStores       *cfgstore.ConfigStores
	bufferedLogHandler *testutil.BufferedLogHandler
	bufferedWriter     *testutil.BufferedWriter
	logger             *slog.Logger
	dirsProvider       *cfgstore.DirsProvider
}

// TestServer represents a running test server instance
type TestServer struct {
	BaseURL     string
	Port        int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	serverError error
	env         *testEnvironment
	t           *testing.T
}

// =============================================================================
// RFC 9457 Assertion Helper
// =============================================================================

// assertRFC9457Equal compares two Response structs field by field
// with clear error messages for each difference.
func assertRFC9457Equal(t *testing.T, got, want *rfc9457.Response) {
	t.Helper()

	if got == nil && want == nil {
		return
	}
	if got == nil {
		t.Error("Got nil Response, want non-nil")
		return
	}
	if want == nil {
		t.Error("Want nil Response, got non-nil")
		return
	}

	// Verify all RFC 9457 fields exhaustively
	if got.Type != want.Type {
		t.Errorf("Type: got '%s', want '%s'", got.Type, want.Type)
	}
	if got.Title != want.Title {
		t.Errorf("Title: got '%s', want '%s'", got.Title, want.Title)
	}
	if got.Status != want.Status {
		t.Errorf("Status: got %d, want %d", got.Status, want.Status)
	}
	if got.Detail != want.Detail {
		t.Errorf("Detail: got '%s', want '%s'", got.Detail, want.Detail)
	}
	if got.Instance != want.Instance {
		t.Errorf("Instance: got '%s', want '%s'", got.Instance, want.Instance)
	}
	if len(got.Extensions) == 0 {
		t.Error("Extension not set")
		return
	}
	ext := got.Extensions[0].(apiresp.RFC9457Extension)

	if len(want.Extensions) == 0 {
		t.Error("Want extension not set")
		return
	}
	wantExt := want.Extensions[0].(apiresp.RFC9457Extension)
	if ext.Parameter != wantExt.Parameter {
		t.Errorf("Parameter: ext '%s', wantExt.'%s'", ext.Parameter, wantExt.Parameter)
	}
	if ext.ExpectedType != wantExt.ExpectedType {
		t.Errorf("ExpectedType: ext '%s', wantExt.'%s'", ext.ExpectedType, wantExt.ExpectedType)
	}
	if ext.ReceivedValue != wantExt.ReceivedValue {
		t.Errorf("ReceivedValue: ext '%s', wantExt.'%s'", ext.ReceivedValue, wantExt.ReceivedValue)
	}
	if ext.Location != wantExt.Location {
		t.Errorf("Location: ext '%s', wantExt.'%s'", ext.Location, wantExt.Location)
	}
	if ext.Suggestion != wantExt.Suggestion {
		t.Errorf("Suggestion: ext '%s', wantExt.'%s'", ext.Suggestion, wantExt.Suggestion)
	}

	// Compare Constraint (handles nil and any type)
	if wantExt.Constraint != nil {
		if ext.Constraint != wantExt.Constraint {
			t.Errorf("Constraint: ext '%v', wantExt.'%v'", ext.Constraint, wantExt.Constraint)
		}
	} else if ext.Constraint != nil {
		t.Errorf("Constraint: ext '%v', wantExt.nil", ext.Constraint)
	}

	// Compare ValidationErrors slice
	if len(ext.ValidationErrors) != len(wantExt.ValidationErrors) {
		t.Errorf("ValidationErrors length: ext %d, wantExt.%d", len(ext.ValidationErrors), len(wantExt.ValidationErrors))
		return
	}
	for i := range wantExt.ValidationErrors {
		extErr := ext.ValidationErrors[i]
		wantExtErr := wantExt.ValidationErrors[i]
		if extErr.Parameter != wantExtErr.Parameter {
			t.Errorf("ValidationErrors[%d].Parameter: ext '%s', wantExt.'%s'", i, extErr.Parameter, wantExtErr.Parameter)
		}
		if extErr.Location != wantExtErr.Location {
			t.Errorf("ValidationErrors[%d].Location: ext '%s', wantExt.'%s'", i, extErr.Location, wantExtErr.Location)
		}
		if extErr.Expected != wantExtErr.Expected {
			t.Errorf("ValidationErrors[%d].Expected: ext '%s', wantExt.'%s'", i, extErr.Expected, wantExtErr.Expected)
		}
		if extErr.Received != wantExtErr.Received {
			t.Errorf("ValidationErrors[%d].Received: ext '%s', wantExt.'%s'", i, extErr.Received, wantExtErr.Received)
		}
		if extErr.Message != wantExtErr.Message {
			t.Errorf("ValidationErrors[%d].Message: ext '%s', wantExt.'%s'", i, extErr.Message, wantExtErr.Message)
		}
	}
}

// =============================================================================
// Test Environment Setup
// =============================================================================

// setupTestEnvironment creates an isolated test environment for a single test case
func setupTestEnvironment(t *testing.T, configContent string) *testEnvironment {
	t.Helper()

	appInfo := xmluisvr.AppInfo()

	// Create ONE root fixture for this test environment
	rootFix := fsfix.NewRootFixture(fmt.Sprintf("%s-api", appInfo.AppSlug()))

	args := &cstest.TestDirsProviderArgs{
		Username:   TestUsername,
		ProjectDir: dt.DirPath(localsvr.AppSlug),
		ConfigSlug: localsvr.ConfigSlug,
		TestRootFunc: func() dt.DirPath {
			return rootFix.Dir()
		},
	}

	dirsProvider := cstest.NewTestDirsProvider(args)
	// Get the config stores map
	css := cfgstore.NewConfigStores(cfgstore.ConfigStoresArgs{
		ConfigStoreArgs: cfgstore.ConfigStoreArgs{
			ConfigSlug:   localsvr.ConfigSlug,
			RelFilepath:  localsvr.ConfigFile,
			DirsProvider: dirsProvider,
		},
	})
	cliStore := css.CLIConfigStore()
	cliDir, err := cstest.GetRelConfigDir(cliStore, args)
	if err != nil {
		t.Fatal(err)
	}
	// Create .config directory for user config
	cliFix := rootFix.AddDirFixture(t, cliDir, nil)
	configFile := cliFix.AddFileFixture(t, localsvr.ConfigFile, &fsfix.FileFixtureArgs{
		Content: configContent,
	})
	dbFix := cliFix.AddDirFixture(t, "db", nil)
	sqlite3Fix := dbFix.AddDirFixture(t, "sqlite3", nil)
	// Create bootstrap file in the test data directory
	bootstrapFile := sqlite3Fix.AddFileFixture(t, "bootstrap.sql", &fsfix.FileFixtureArgs{
		Content: BootstrapSQL(),
	})

	projectStore := css.ProjectConfigStore()
	projectDir, err := cstest.GetRelConfigDir(projectStore, args)
	if err != nil {
		t.Fatal(err)
	}
	// Create project directory for project config (same content for simplicity)
	projectFix := rootFix.AddDirFixture(t, projectDir, nil)
	projectFix.AddFileFixture(t, localsvr.ConfigFile, &fsfix.FileFixtureArgs{
		Content: configContent,
	})

	// Create the fixture directory structure on disk
	rootFix.Create(t)

	css.CLIConfigStore().SetConfigDir(cliFix.Dir())
	css.ProjectConfigStore().SetConfigDir(projectFix.Dir())

	// Generate unique database path
	dbPath := dt.FilepathJoin(rootFix.Dir(), "test.db")

	// Setup buffered logger and CLI writer
	logger := testutil.GetBufferedLogger()
	handler := testutil.GetBufferedLogHandler()
	bufferedWriter := testutil.NewBufferedWriter()

	// Set as global logger
	localsvr.SetLogger(logger)

	return &testEnvironment{
		rootFixture:        rootFix,
		appInfo:            appInfo,
		dbPath:             dbPath,
		bootstrapFile:      bootstrapFile,
		configFile:         configFile,
		configStores:       css,
		bufferedLogHandler: handler,
		bufferedWriter:     bufferedWriter,
		dirsProvider:       dirsProvider,
		logger:             logger,
	}
}

// cleanup cleans up the test environment
func (env *testEnvironment) cleanup(t *testing.T) {
	t.Helper()
	env.rootFixture.Cleanup()
	// Remove database file if it exists
	_, err := env.dbPath.Stat()
	if err != nil {
		goto end
	}
	err = env.dbPath.Remove()
	if err != nil {
		t.Errorf("Warning: failed to remove database file: %v", err)
	}
end:
	return
}

// =============================================================================
// Test Server Management
// =============================================================================

// setupTestServer creates and starts a test server instance
func setupTestServer(t *testing.T, configContent string) *TestServer {
	t.Helper()

	env := setupTestEnvironment(t, configContent)

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	// Configure options
	cfgOpts := &cfgldr.Options{
		HTTPPort:        port,
		ConnectString:   string(env.dbPath),
		DBBootstrapFile: string(env.bootstrapFile.Filepath),
		Timeout:         300,
		Verbosity:       3, // Max verbosity for debugging
		ErrorStyle:      string(localsvr.DevelopmentStyle),
	}

	// Load root config
	rootConfig, err := cfgldr.LoadRootConfigV1(cfgldr.LoadRootConfigV1Args{
		AppInfo:      env.appInfo,
		Options:      cfgOpts,
		ConfigStores: env.configStores,
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	options, err := xmluisvr.ParseOptions(cfgOpts)
	if err != nil {
		t.Fatalf("Failed to parse options: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := xmluisvr.ParseConfig(ctx, rootConfig, xmluisvr.ParseConfigArgs{
		Options:      options,
		Logger:       env.logger,
		Writer:       env.bufferedWriter,
		DirsProvider: env.dirsProvider,
	})
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Setup run arguments
	runArgs := &xmluisvr.RunArgs{
		CLIArgs: []string{},
		AppInfo: env.appInfo,
		Options: options,
		Config:  config,
	}

	// Start server in goroutine
	var serverError error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		serverError = xmluisvr.Run(ctx, runArgs)
		if serverError != nil {
			t.Errorf("Server error: %v", serverError)
		}
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Wait for server to be ready by polling health endpoint
	if !waitForServerReady(t, baseURL, 10*time.Second) {
		cancel()
		wg.Wait()
		if serverError != nil {
			t.Errorf("Server failed with error: %v", serverError)
		}
		// Print logs to help diagnose
		logEntries, _ := env.bufferedLogHandler.GetLogEntries()
		if len(logEntries) > 0 {
			t.Errorf("Server logs (%d entries):", len(logEntries))
			for i, entry := range logEntries {
				t.Errorf("  [%d] %v", i+1, entry)
			}
		}
		// t.Logf("Server output:\n%s", env.bufferedWriter.GetAllOutput())
		t.Fatalf("Server did not become ready within timeout")
	}

	server := &TestServer{
		BaseURL:     baseURL,
		Port:        port,
		ctx:         ctx,
		cancel:      cancel,
		wg:          &wg,
		serverError: serverError,
		env:         env,
		t:           t,
	}

	return server
}

// Cleanup shuts down the server and cleans up resources
func (s *TestServer) Cleanup() {
	s.t.Helper()

	// Stop server
	s.cancel()

	// Wait for graceful shutdown
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Server shut down gracefully
	case <-time.After(5 * time.Second):
		// s.t.Log("Warning: Server did not shut down gracefully within 5 seconds")
	}

	// Check for server errors (ignore context cancellation)
	if s.serverError != nil && !strings.Contains(s.serverError.Error(), "context canceled") {
		s.t.Errorf("Server error: %v", s.serverError)
	}

	// Cleanup environment
	s.env.cleanup(s.t)
}

// =============================================================================
// Test Request Execution
// =============================================================================

// makeHTTPRequest creates and executes an HTTP request
func makeHTTPRequest(baseURL, method, path, body string) (resp *http.Response, respBody []byte, err error) {
	var httpReq *http.Request
	var client *http.Client

	url := baseURL + path

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	httpReq, err = http.NewRequest(method, url, reqBody)
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		goto end
	}

	if body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	client = &http.Client{Timeout: 300 * time.Second}
	resp, err = client.Do(httpReq)
	if err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		goto end
	}

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		dt.CloseOrLog(resp.Body)
		err = fmt.Errorf("failed to read response body: %w", err)
		goto end
	}
end:
	return resp, respBody, err
}

// =============================================================================
// Utility Functions
// =============================================================================

// findAvailablePort finds an available port for testing
func findAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer dt.LogOnError(listen.Close())

	port := listen.Addr().(*net.TCPAddr).Port
	return port, nil
}

// waitForServerReady polls the server's /healthz endpoint until it responds or timeout occurs
func waitForServerReady(t *testing.T, baseURL string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	healthURL := baseURL + "/healthz"

	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			err := resp.Body.Close()
			if err != nil {
				t.Errorf("Failed to close body for %s: %v", healthURL, err)
			}
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// closeOrError closes an io.Closer and fails the test if there's an error
func closeOrError(t *testing.T, closer io.Closer) {
	t.Helper()
	if err := closer.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

// SetupTestServer sets up a test server using the  test config.
// This is a convenience wrapper that loads api__test.json which contains
// all API endpoints for integration testing (data types, constraints, parameters, etc.).
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Read the  test config that has all endpoints defined
	configBytes, err := os.ReadFile("./test-data/api__test.json")
	if err != nil {
		t.Fatalf("Failed to read  test config: %v", err)
	}

	// Use the TestEnv constant for type safety and documentation
	return setupTestServer(t, string(configBytes))
}

// runTestRequest executes a test request and validates the response
func runTestRequest(t *testing.T, baseURL string, req testRequest) {
	t.Helper()

	resp, body, err := makeHTTPRequest(baseURL, req.method, req.path, req.body)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer closeOrError(t, resp.Body)

	// Check status code
	if resp.StatusCode != req.expectedStatus {
		t.Errorf("Status code: got %d, want %d\nBody: %s", resp.StatusCode, req.expectedStatus, string(body))
	}

	// If we expect an RFC9457 error response, validate it
	if req.expectedRFC9457 != nil {
		var got rfc9457.Response
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("Failed to unmarshal RFC9457 response: %v\nBody: %s", err, string(body))
		}
		assertRFC9457Equal(t, &got, req.expectedRFC9457)
		return
	}

	// Check expected fields in JSON response
	if len(req.expectedFields) > 0 {
		bodyStr := string(body)

		// Try to unmarshal as array first (cardinality="many")
		var resultArray []map[string]interface{}
		if err := json.Unmarshal(body, &resultArray); err == nil && len(resultArray) > 0 {
			// Validate fields exist in first element of array
			for _, field := range req.expectedFields {
				if _, exists := resultArray[0][field]; !exists {
					t.Errorf("Expected field '%s' not found in response: %s", field, bodyStr)
				}
			}
		} else {
			// Try as single object (cardinality="one")
			var resultObject map[string]interface{}
			if err := json.Unmarshal(body, &resultObject); err != nil {
				t.Fatalf("Failed to unmarshal JSON response as object or array: %v\nBody: %s", err, bodyStr)
			}
			// Validate fields exist in object
			for _, field := range req.expectedFields {
				if _, exists := resultObject[field]; !exists {
					t.Errorf("Expected field '%s' not found in response: %s", field, bodyStr)
				}
			}
		}
	}

	// Check shouldContain (legacy)
	bodyStr := string(body)
	for _, contains := range req.shouldContain {
		if !strings.Contains(bodyStr, contains) {
			t.Errorf("Response should contain '%s' but doesn't.\nBody: %s", contains, bodyStr)
		}
	}

	// Check shouldNotHave
	for _, notHave := range req.shouldNotHave {
		if strings.Contains(bodyStr, notHave) {
			t.Errorf("Response should NOT contain '%s' but does.\nBody: %s", notHave, bodyStr)
		}
	}
}
