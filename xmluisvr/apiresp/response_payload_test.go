package apiresp_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mikeschinkel/go-rfc9457"
	"github.com/mikeschinkel/go-testutil"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apiresp"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// TestMain sets up package-level configuration before running tests
func TestMain(m *testing.M) {
	// Set the GitHub repo URL to avoid log.Fatal during tests
	apiresp.SetGitHubRepoURL(localsvr.GitHubRepoURL)

	// Set up the buffered logger for tests
	localsvr.SetLogger(testutil.GetBufferedLogger())

	// Run tests
	os.Exit(m.Run())
}

// TestInternalServerErrorPayload tests the InternalServerErrorPayload function
func TestInternalServerErrorPayload(t *testing.T) {
	var resp *rfc9457.Response

	req := httptest.NewRequest("GET", "/api/users/abc", nil)

	pr := apiresp.InternalServerErrorPayload(req, apiresp.PayloadArgs{})

	// Should return Response
	if !errors.As(pr.ResponsePayload, &resp) {
		t.Fatalf("Expected *apiresp.Response, got %T", pr.ResponsePayload)
	}

	// Verify all fields
	if resp.Type != rfc9457.InternalServerErrorType {
		t.Errorf("Type: got %v, want %v", resp.Type, rfc9457.InternalServerErrorType)
	}
	if resp.Title != "Internal Server Error" {
		t.Errorf("Title: got %q, want %q", resp.Title, "Internal Server Error")
	}
	if resp.Status != http.StatusInternalServerError {
		t.Errorf("Status: got %d, want %d", resp.Status, http.StatusInternalServerError)
	}
	if resp.Detail == "" {
		t.Error("Detail should not be empty")
	}
	if resp.Instance != req.RequestURI {
		t.Errorf("Instance: got %q, want %q", resp.Instance, req.RequestURI)
	}

	if len(resp.Extensions) != 1 {
		t.Errorf("Number of extensions should be 1, got %d", len(resp.Extensions))
	}
	respExt := resp.Extensions[0]

	ext, ok := respExt.(apiresp.RFC9457Extension)
	if !ok {
		t.Errorf("Extensions is not of type apiresp.RFC9457Extension, got %T instead", respExt)
	}

	// Optional fields should be empty
	if ext.Parameter != "" {
		t.Errorf("Parameter should be empty, got %q", ext.Parameter)
	}
	if ext.ExpectedType != "" {
		t.Errorf("ExpectedType should be empty, got %q", ext.ExpectedType)
	}
	if ext.ReceivedValue != "" {
		t.Errorf("ReceivedValue should be empty, got %q", ext.ReceivedValue)
	}
	if ext.Location != "" {
		t.Errorf("Location should be empty, got %q", ext.Location)
	}
	if ext.Constraint != nil {
		t.Errorf("Constraint should be nil, got %v", ext.Constraint)
	}
	if ext.Suggestion != "" {
		t.Errorf("Suggestion should be empty, got %q", ext.Suggestion)
	}
	if len(ext.ValidationErrors) != 0 {
		t.Errorf("ValidationErrors should be empty, got %d items", len(ext.ValidationErrors))
	}
}

// TestEndpointNotMatchedPayload tests the EndpointNotMatchedPayload function
func TestEndpointNotMatchedPayload(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/nonexistent", nil)

	pr := apiresp.EndpointNotMatchedPayload(req, apiresp.PayloadArgs{})
	// Should return Response

	var resp *rfc9457.Response
	if !errors.As(pr.ResponsePayload, &resp) {
		t.Fatalf("Expected *apiresp.Response, got %T", pr.ResponsePayload)
	}

	// Verify all fields
	if resp.Type != rfc9457.EndpointNotMatchedErrorType {
		t.Errorf("Type: got %v, want %v", resp.Type, rfc9457.EndpointNotMatchedErrorType)
	}
	if resp.Title != "Endpoint Not Matched" {
		t.Errorf("Title: got %q, want %q", resp.Title, "Endpoint Not Matched")
	}
	if resp.Status != http.StatusNotFound {
		t.Errorf("Status: got %d, want %d", resp.Status, http.StatusNotFound)
	}
	if resp.Detail == "" {
		t.Error("Detail should not be empty")
	}
	if resp.Instance != req.RequestURI {
		t.Errorf("Instance: got %q, want %q", resp.Instance, req.RequestURI)
	}

	if len(resp.Extensions) != 1 {
		t.Errorf("Number of extensions should be 1, got %d", len(resp.Extensions))
	}
	respExt := resp.Extensions[0]

	ext, ok := respExt.(apiresp.RFC9457Extension)
	if !ok {
		t.Errorf("Extensions is not of type apiresp.RFC9457Extension, got %T instead", respExt)
	}

	// Optional fields should be empty
	if ext.Parameter != "" {
		t.Errorf("Parameter should be empty, got %q", ext.Parameter)
	}
	if ext.ExpectedType != "" {
		t.Errorf("ExpectedType should be empty, got %q", ext.ExpectedType)
	}
	if ext.ReceivedValue != "" {
		t.Errorf("ReceivedValue should be empty, got %q", ext.ReceivedValue)
	}
	if ext.Location != "" {
		t.Errorf("Location should be empty, got %q", ext.Location)
	}
	if ext.Constraint != nil {
		t.Errorf("Constraint should be nil, got %v", ext.Constraint)
	}
	if ext.Suggestion != "" {
		t.Errorf("Suggestion should be empty, got %q", ext.Suggestion)
	}
	if len(ext.ValidationErrors) != 0 {
		t.Errorf("ValidationErrors should be empty, got %d items", len(ext.ValidationErrors))
	}
}

// TestUnprocessableEntityPayload_WithRFC9457 tests UnprocessableEntityPayload with valid RFC9457
func TestUnprocessableEntityPayload_WithRFC9457(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users/abc", nil)

	inputRFC9457 := &rfc9457.Response{
		Type:     rfc9457.InvalidParameterErrorType,
		Title:    "Invalid Parameter Type",
		Status:   http.StatusUnprocessableEntity,
		Detail:   "Parameter 'id' expected type 'int' but received 'abc'",
		Instance: "/api/users/abc",
		Extensions: []rfc9457.Extension{
			apiresp.RFC9457Extension{
				Parameter:     "id",
				ExpectedType:  "int",
				ReceivedValue: "abc",
				Location:      apiresp.PathLocation,
			},
		},
	}

	pr := apiresp.UnprocessableEntityPayload(req, apiresp.PayloadArgs{
		RFC9457: inputRFC9457,
	})

	// Should return the same Response
	var resp *rfc9457.Response
	if !errors.As(pr.ResponsePayload, &resp) {
		t.Fatalf("Expected *apiresp.Response, got %T", pr.ResponsePayload)
	}

	// Should be the exact same instance
	//goland:noinspection GoDirectComparisonOfErrors
	if resp != inputRFC9457 {
		t.Error("UnprocessableEntityPayload should return the same Response instance")
	}

	// Verify it's unchanged
	if resp.Type != rfc9457.InvalidParameterErrorType {
		t.Errorf("Type: got %v, want %v", resp.Type, rfc9457.InvalidParameterErrorType)
	}
	if resp.Title != "Invalid Parameter Type" {
		t.Errorf("Title: got %q, want %q", resp.Title, "Invalid Parameter Type")
	}
	if resp.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status: got %d, want %d", resp.Status, http.StatusUnprocessableEntity)
	}

	if len(resp.Extensions) != 1 {
		t.Errorf("Number of extensions should be 1, got %d", len(resp.Extensions))
	}
	respExt := resp.Extensions[0]

	ext, ok := respExt.(apiresp.RFC9457Extension)
	if !ok {
		t.Errorf("Extensions is not of type apiresp.RFC9457Extension, got %T instead", respExt)
	}

	if ext.Parameter != "id" {
		t.Errorf("Parameter: got %q, want %q", ext.Parameter, "id")
	}
}

// TestUnprocessableEntityPayload_WithNil tests UnprocessableEntityPayload with nil RFC9457
func TestUnprocessableEntityPayload_WithNil(t *testing.T) {
	var resp *rfc9457.Response

	req := httptest.NewRequest("GET", "/api/users/abc", nil)

	pr := apiresp.UnprocessableEntityPayload(req, apiresp.PayloadArgs{})

	// Should return InternalServerError instead
	if !errors.As(pr.ResponsePayload, &resp) {
		t.Fatalf("Expected *apiresp.Response, got %T", pr.ResponsePayload)
	}

	// Should be an internal server error (fallback behavior)
	if resp.Type != rfc9457.InternalServerErrorType {
		t.Errorf("Type: got %v, want %v", resp.Type, rfc9457.InternalServerErrorType)
	}
	if resp.Status != http.StatusInternalServerError {
		t.Errorf("Status: got %d, want %d", resp.Status, http.StatusInternalServerError)
	}
	if resp.Title != "Internal Server Error" {
		t.Errorf("Title: got %q, want %q", resp.Title, "Internal Server Error")
	}
}
