package cfgldr_test

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"os"
	"testing"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/stretchr/testify/require"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
)

// TestDirectJSONLoad tests loading the config directly from JSON without cfgstore
// to isolate whether the issue is in JSON unmarshaling or in cfgstore merging.
func TestDirectJSONLoad(t *testing.T) {
	// Load the project config file that has 5 endpoints
	data, err := os.ReadFile("./test-data/project-config.localsvr.json")
	require.NoError(t, err, "Failed to read project config file")

	var rc cfgldr.RootConfigV1
	err = jsonv2.Unmarshal(data, &rc)
	require.NoError(t, err, "Failed to unmarshal JSON")

	// Check that we have the server config
	require.NotNil(t, rc.ServerConfig, "ServerConfig should not be nil")

	// Check that we have the API config
	api := rc.APIConfig()
	require.NotNil(t, api, "API config should not be nil")

	apiV2, ok := api.(*cfgldr.APIConfigV2)
	require.True(t, ok, "API config should be v2")

	// CRITICAL: Check that endpoints were loaded from JSON
	t.Logf("API Name: %s", apiV2.Name)
	t.Logf("Webroot: %s", apiV2.Webroot)
	t.Logf("Number of endpoints BEFORE Normalize: %d", len(apiV2.Endpoints))

	require.Equal(t, "User-definable XMLUI Local Server API", apiV2.Name, "API name mismatch")
	require.Equal(t, "./webroot", apiV2.Webroot, "Webroot mismatch")
	require.Len(t, apiV2.Endpoints, 5, "Should have 5 endpoints from JSON")

	// Show what we loaded
	for i, ep := range apiV2.Endpoints {
		t.Logf("Endpoint %d: %s %s", i, ep.Method, ep.Path)
	}

	// Now test what happens after Normalize
	opts := cfgldr.NewOptions(cfgldr.OptionsArgs{})
	err = rc.Normalize(cfgstore.NormalizeArgs{
		DirType:    cfgstore.CLIConfigDirType,
		SourceFile: "./test-data/project-config.localsvr.json",
		Options:    opts,
	})
	require.NoError(t, err, "Normalize should not error")

	// Check again after normalize
	apiAfter := rc.APIConfig()
	require.NotNil(t, apiAfter, "API config should not be nil after Normalize")

	apiV2After, ok := apiAfter.(*cfgldr.APIConfigV2)
	require.True(t, ok, "API config should still be v2 after Normalize")

	t.Logf("API Name AFTER Normalize: %s", apiV2After.Name)
	t.Logf("Webroot AFTER Normalize: %s", apiV2After.Webroot)
	t.Logf("Number of endpoints AFTER Normalize: %d", len(apiV2After.Endpoints))

	// CRITICAL: Endpoints should NOT disappear after Normalize!
	require.Len(t, apiV2After.Endpoints, 5, "Should STILL have 5 endpoints after Normalize")

	// Show the full config as JSON for debugging
	bytes, _ := jsonv2.Marshal(&rc, jsontext.WithIndent("  "))
	t.Logf("Full config after Normalize:\n%s", string(bytes))
}
