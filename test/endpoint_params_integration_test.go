package test

import (
	jsonv2 "encoding/json/v2"
	"errors"
	"strings"
	"testing"

	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

// fullAPIConfig represents a complete API configuration for integration testing
type fullAPIConfig struct {
	Version   string                 `json:"version"`
	BasePath  string                 `json:"base_path"`
	Endpoints []cfgldr.APIEndpointV2 `json:"endpoints"`
}

func TestEndpointParams_FullConfigIntegration(t *testing.T) {
	// This represents a realistic API configuration with mixed parameter formats
	configJSON := `{
		"version": "2.0",
		"base_path": "/api/v1",
		"endpoints": [
			{
				"endpoint": "GET /tasks/search/{project_id:int}",
				"description": "Search tasks within a given project (path param project_id + query-string param q)",
				"query": "SELECT t.id, t.title, t.status, t.priority, IFNULL(au.email,'') AS assignee_email FROM tasks t LEFT JOIN users au ON au.id = t.assignee_id WHERE t.project_id = :project_id AND (LOWER(t.title) LIKE LOWER('%' || :q || '%') OR LOWER(t.details) LIKE LOWER('%' || :q || '%')) ORDER BY t.priority DESC, t.id;",
				"params": {
					"q": "string",
					"sort": "string:enum[asc,desc]",
					"limit": "int:range[1..50]",
					"@note": ["Just a little bit of info", "for posterity"]
				},
				"cardinality": "many",
				"row_type": "columns",
				"column_types": ["integer", "string", "string", "integer", "string"]
			},
			{
				"endpoint": "GET /tasks/by-project/{project:string}",
				"description": "Tasks for a project using project in the path and owner email as a query-string parameter",
				"query": "SELECT t.id, t.title, t.status, t.priority, t.due_date, au.email AS assignee_email, au.name AS assignee_name, t.created_at FROM tasks t JOIN projects p ON p.id = t.project_id JOIN users ou ON ou.id = p.owner_id LEFT JOIN users au ON au.id = t.assignee_id WHERE ou.email = :email AND p.name = :project ORDER BY t.priority DESC, t.created_at;",
				"params": [
					{
						"name": "email",
						"type": "string"
					}
				],
				"cardinality": "many",
				"row_type": "columns",
				"column_types": [
					"integer",
					"string",
					"string",
					"integer",
					"string?",
					"string?",
					"string?",
					"string"
				]
			},
			{
				"endpoint": "GET /users/{id:int}",
				"description": "Get a single user by numeric id (path parameter only)",
				"query": "SELECT id, email, name, created_at FROM users WHERE id = :id;",
				"cardinality": "one",
				"row_type": "columns",
				"column_types": ["integer", "string", "string", "string"]
			},
			{
				"endpoint": "GET /hello",
				"description": "Hello World Endpoint",
				"query": "SELECT 'Hello World';",
				"params": [],
				"cardinality": "one",
				"row_type": "string",
				"column_types": []
			},
			{
				"endpoint": "GET /projects/by-owner/{email:string}",
				"description": "Projects owned by a given user (owner email as a path parameter)",
				"query": "SELECT p.id, p.name, p.status, p.created_at FROM projects p WHERE p.owner_id = (SELECT id FROM users WHERE email = :email) ORDER BY p.created_at DESC;",
				"cardinality": "many",
				"row_type": "columns",
				"column_types": ["integer", "string", "string", "string"]
			}
		]
	}`

	var config fullAPIConfig
	err := jsonv2.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal full config: %v", err)
	}

	// Verify we have all expected endpoints
	if len(config.Endpoints) != 5 {
		t.Fatalf("Expected 5 endpoints, got %d", len(config.Endpoints))
	}

	t.Run("tasks_search_endpoint_map_params", func(t *testing.T) {
		endpoint := config.Endpoints[0]

		// Should have 3 real params (comments filtered out)
		params, ok := endpoint.Params.(cfgldr.APIParamsV1)
		if !ok {
			t.Fatalf("Expected APIParamsV1, got %T", endpoint.Params)
		}
		if len(params) != 3 {
			t.Errorf("Expected 3 params, got %d", len(params))
		}

		// Verify specific params exist
		paramMap := make(map[string]cfgldr.APIParamV1)
		for _, p := range params {
			paramMap[p.NameSpec] = p
		}

		if p, exists := paramMap["q"]; !exists {
			t.Error("Missing 'q' parameter")
		} else if p.Type != "string" {
			t.Errorf("Expected q type 'string', got %q", p.Type)
		}

		if p, exists := paramMap["sort"]; !exists {
			t.Error("Missing 'sort' parameter")
		} else if p.Constraints != "enum[asc,desc]" {
			t.Errorf("Expected sort constraint 'enum[asc,desc]', got %q", p.Constraints)
		}

		if p, exists := paramMap["limit"]; !exists {
			t.Error("Missing 'limit' parameter")
		} else if p.Constraints != "range[1..50]" {
			t.Errorf("Expected limit constraint 'range[1..50]', got %q", p.Constraints)
		}
	})

	t.Run("tasks_by_project_endpoint_array_params", func(t *testing.T) {
		endpoint := config.Endpoints[1]

		// Should have 1 param
		params, ok := endpoint.Params.(cfgldr.APIParamsV1)
		if !ok {
			t.Fatalf("Expected APIParamsV1, got %T", endpoint.Params)
		}
		if len(params) != 1 {
			t.Errorf("Expected 1 param, got %d", len(params))
		}

		if params[0].NameSpec != "email" {
			t.Errorf("Expected param name 'email', got %q", params[0].NameSpec)
		}
		if params[0].Type != "string" {
			t.Errorf("Expected param type 'string', got %q", params[0].Type)
		}
	})

	t.Run("simple_endpoint_no_explicit_params", func(t *testing.T) {
		endpoint := config.Endpoints[2] // GET /users/{id:int}

		// Should have empty params
		params, ok := endpoint.Params.(cfgldr.APIParamsV1)
		if !ok {
			t.Fatalf("Expected APIParamsV1, got %T", endpoint.Params)
		}
		if len(params) != 0 {
			t.Errorf("Expected 0 params, got %d", len(params))
		}
	})

	t.Run("hello_endpoint_empty_array_params", func(t *testing.T) {
		endpoint := config.Endpoints[3] // GET /hello

		params, ok := endpoint.Params.(cfgldr.APIParamsV1)
		if !ok {
			t.Fatalf("Expected APIParamsV1, got %T", endpoint.Params)
		}
		if len(params) != 0 {
			t.Errorf("Expected 0 params, got %d", len(params))
		}
	})
}

func TestEndpointParams_RoundTripMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "map_format_with_comments",
			json: `{
				"endpoint": "GET /test/{id:int}",
				"description": "Test endpoint",
				"query": "SELECT * FROM test WHERE id = :id",
				"params": {
					"limit": "int:range[1..100]",
					"@note": ["Multi-line", "comment"],
					"sort": "string:enum[asc,desc]"
				},
				"cardinality": "many",
				"row_type": "columns",
				"column_types": ["integer", "string"]
			}`,
		},
		{
			name: "array_format",
			json: `{
				"endpoint": "POST /users",
				"description": "Create user",
				"query": "INSERT INTO users (email, name) VALUES (:email, :name)",
				"params": [
					{"name": "email", "type": "string", "constraints": ""},
					{"name": "name", "type": "string", "constraints": "length[1..100]"}
				],
				"cardinality": "one",
				"row_type": "json"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First unmarshal
			var endpoint1 cfgldr.APIEndpointV2
			err := jsonv2.Unmarshal([]byte(tt.json), &endpoint1)
			if err != nil {
				t.Fatalf("First unmarshal failed: %v", err)
			}

			// Marshal back to JSON
			marshaled, err := jsonv2.Marshal(endpoint1)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal again
			var endpoint2 cfgldr.APIEndpointV2
			err = jsonv2.Unmarshal(marshaled, &endpoint2)
			if err != nil {
				t.Fatalf("Second unmarshal failed: %v", err)
			}

			// Compare key fields
			if endpoint1.Endpoint() != endpoint2.Endpoint() {
				t.Errorf("Endpoint mismatch: %q vs %q", endpoint1.Endpoint(), endpoint2.Endpoint())
			}
			if endpoint1.Description != endpoint2.Description {
				t.Errorf("Description mismatch: %q vs %q", endpoint1.Description, endpoint2.Description)
			}
			if endpoint1.Query != endpoint2.Query {
				t.Errorf("Query mismatch: %q vs %q", endpoint1.Query, endpoint2.Query)
			}

			// Compare params (both should be APIParamsV1 after unmarshaling)
			params1, ok1 := endpoint1.Params.(cfgldr.APIParamsV1)
			params2, ok2 := endpoint2.Params.(cfgldr.APIParamsV1)
			if !ok1 || !ok2 {
				t.Fatalf("Parameters type mismatch: %T vs %T", endpoint1.Params, endpoint2.Params)
			}
			if len(params1) != len(params2) {
				t.Fatalf("Parameters length mismatch: %d vs %d", len(params1), len(params2))
			}

			// Compare individual params (order might differ for map format, so check by name)
			paramMap1 := make(map[string]cfgldr.APIParamV1)
			paramMap2 := make(map[string]cfgldr.APIParamV1)
			for _, p := range params1 {
				paramMap1[p.NameSpec] = p
			}
			for _, p := range params2 {
				paramMap2[p.NameSpec] = p
			}

			if len(paramMap1) != len(paramMap2) {
				t.Errorf("Param count mismatch: %d vs %d", len(paramMap1), len(paramMap2))
			}

			for name, p1 := range paramMap1 {
				p2, exists := paramMap2[name]
				if !exists {
					t.Errorf("Param %q missing in second endpoint", name)
					continue
				}
				if p1.Type != p2.Type {
					t.Errorf("Param %q type mismatch: %q vs %q", name, p1.Type, p2.Type)
				}
				if p1.Constraints != p2.Constraints {
					t.Errorf("Param %q constraints mismatch: %q vs %q", name, p1.Constraints, p2.Constraints)
				}
			}
		})
	}
}

func TestEndpointParams_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantErr    error
		wantAnyErr bool // true if we just want any error, not a specific sentinel
	}{
		{
			name: "invalid_param_map_nested_object",
			json: `{
				"endpoint": "GET /test",
				"params": {
					"bad_param": {"nested": "object"}
				}
			}`,
			wantErr: cfgldr.ErrAPIParamsMapCannotBeNested,
		},
		{
			name: "invalid_param_map_array_for_param",
			json: `{
				"endpoint": "GET /test",
				"params": {
					"bad_param": ["not", "allowed"]
				}
			}`,
			wantErr: cfgldr.ErrAPIParamsMapCannotContainArray,
		},
		{
			name: "invalid_param_type_in_conversion",
			json: `{
				"endpoint": "GET /test",
				"params": {
					"bad_param": "invalid_type:constraint"
				}
			}`,
			wantAnyErr: true, // Error comes from go-pathvars, not cfgldr
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var endpoint cfgldr.APIEndpointV2
			err := jsonv2.Unmarshal([]byte(tt.json), &endpoint)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Expected error %v, got: %v", tt.wantErr, err)
				}
			} else if tt.wantAnyErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEndpointParams_LargeConfigPerformance(t *testing.T) {
	// Generate a larger configuration to test performance characteristics
	var endpoints []string
	for i := 0; i < 100; i++ {
		endpoint := `{
			"endpoint": "GET /resource` + string(rune('A'+i%26)) + `/{id:int}",
			"description": "Get resource ` + string(rune('A'+i%26)) + `",
			"query": "SELECT * FROM resource` + string(rune('A'+i%26)) + ` WHERE id = :id",
			"params": {
				"limit": "int:range[1..100]",
				"offset": "int:range[0..10000]",
				"sort": "string:enum[asc,desc]",
				"@comment": ["Generated endpoint", "for performance testing"]
			},
			"cardinality": "many",
			"row_type": "columns",
			"column_types": ["integer", "string", "string"]
		}`
		endpoints = append(endpoints, endpoint)
	}

	configJSON := `{
		"version": "2.0",
		"base_path": "/api/v1",
		"endpoints": [` + strings.Join(endpoints, ",") + `]
	}`

	var config fullAPIConfig
	err := jsonv2.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal large config: %v", err)
	}

	// Verify all endpoints were parsed correctly
	if len(config.Endpoints) != 100 {
		t.Errorf("Expected 100 endpoints, got %d", len(config.Endpoints))
	}

	// Spot check a few endpoints
	for i, endpoint := range config.Endpoints[:5] {
		params, ok := endpoint.Params.(cfgldr.APIParamsV1)
		if !ok {
			t.Errorf("Endpoint %d: expected APIParamsV1, got %T", i, endpoint.Params)
			continue
		}

		// Should have 3 real params (comments filtered out)
		if len(params) != 3 {
			t.Errorf("Endpoint %d: expected 3 params, got %d", i, len(params))
		}

		// Verify all endpoints have proper parameters parsed
	}
}
