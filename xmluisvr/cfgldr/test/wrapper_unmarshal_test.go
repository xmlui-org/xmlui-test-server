package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
)

// TestWrapperUnmarshalJSON tests if RootConfigV1Wrapper.UnmarshalJSON works correctly
func TestWrapperUnmarshalJSON(t *testing.T) {
	// Load the project config file
	data, err := os.ReadFile("./test-data/project-config.localsvr.json")
	require.NoError(t, err, "Failed to read project config file")

	// Test unmarshaling into the wrapper
	var wrapper cfgldr.RootConfigV1Wrapper
	err = wrapper.UnmarshalJSON(data)
	require.NoError(t, err, "UnmarshalJSON should not error")

	// Check that endpoints were loaded
	api := wrapper.APIConfig()
	require.NotNil(t, api, "API config should not be nil")

	apiV2, ok := api.(*cfgldr.APIConfigV2)
	require.True(t, ok, "API config should be v2")

	// t.Logf("Wrapper loaded %d endpoints", len(apiV2.Endpoints))
	require.Len(t, apiV2.Endpoints, 5, "Wrapper should have 5 endpoints")
}
