package test

import (
	jsonv2 "encoding/json/v2"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

func TestAPIEndpointV2_UnmarshalJSON_ArrayParams(t *testing.T) {
	jsonData := `{
		"method": "GET",
		"path": "/users/{id:int}",
		"description": "Get a single user by ID",
		"query": "SELECT id, email, name FROM users WHERE id = :id",
		"params": [
			{"name": "id", "type": "int", "constraints": ""}
		],
		"cardinality": "one",
		"row_type": "columns",
		"column_types": ["integer", "string", "string"]
	}`

	var endpoint cfgldr.APIEndpointV2
	err := jsonv2.Unmarshal([]byte(jsonData), &endpoint)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Check basic fields
	if endpoint.Endpoint() != "GET /users/{id:int}" {
		t.Errorf("endpoint: got %q, want %q", endpoint.Endpoint(), "GET /users/{id:int}")
	}
	if endpoint.Description != "Get a single user by ID" {
		t.Errorf("description: got %q, want %q", endpoint.Description, "Get a single user by ID")
	}
	if endpoint.Query != "SELECT id, email, name FROM users WHERE id = :id" {
		t.Errorf("query: got %q, want %q", endpoint.Query, "SELECT id, email, name FROM users WHERE id = :id")
	}
	if endpoint.Cardinality != "one" {
		t.Errorf("cardinality: got %q, want %q", endpoint.Cardinality, "one")
	}
	if endpoint.RowType != "columns" {
		t.Errorf("row_type: got %q, want %q", endpoint.RowType, "columns")
	}

	// Check params
	params, ok := endpoint.Params.(cfgldr.APIParamsV1)
	if !ok {
		t.Fatalf("params should be APIParamsV1, got %T", endpoint.Params)
	}
	if len(params) != 1 {
		t.Fatalf("params length: got %d, want 1", len(params))
	}
	if params[0].NameSpec != "id" {
		t.Errorf("param name: got %q, want %q", params[0].NameSpec, "id")
	}
	if params[0].Type != "int" {
		t.Errorf("param type: got %q, want %q", params[0].Type, "int")
	}

	// Check column types
	expectedCols := []string{"integer", "string", "string"}
	if len(endpoint.ColumnTypes) != len(expectedCols) {
		t.Fatalf("column_types length: got %d, want %d", len(endpoint.ColumnTypes), len(expectedCols))
	}
	for i, col := range endpoint.ColumnTypes {
		if col != expectedCols[i] {
			t.Errorf("column_types[%d]: got %q, want %q", i, col, expectedCols[i])
		}
	}
}

func TestAPIEndpointV2_UnmarshalJSON_MapParams(t *testing.T) {
	jsonData := `{
		"method": "GET",
		"path": "/tasks/search/{project_id:int}",
		"description": "Search tasks within a given project",
		"query": "SELECT t.id, t.title FROM tasks t WHERE t.project_id = :project_id",
		"params": {
			"q": "string",
			"sort": "string:enum[asc,desc]",
			"limit": "int:range[1..50]",
			"@note": ["Just a little bit of info", "for posterity"]
		},
		"cardinality": "many",
		"row_type": "columns",
		"column_types": ["integer", "string"]
	}`

	var endpoint cfgldr.APIEndpointV2
	err := jsonv2.Unmarshal([]byte(jsonData), &endpoint)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Check basic fields
	if endpoint.Endpoint() != "GET /tasks/search/{project_id:int}" {
		t.Errorf("endpoint: got %q, want %q", endpoint.Endpoint(), "GET /tasks/search/{project_id:int}")
	}
	if endpoint.Cardinality != "many" {
		t.Errorf("cardinality: got %q, want %q", endpoint.Cardinality, "many")
	}

	// Check params - should be converted to APIParamsV1 but paramsType should reflect map origin
	params, ok := endpoint.Params.(cfgldr.APIParamsV1)
	if !ok {
		t.Fatalf("params should be APIParamsV1, got %T", endpoint.Params)
	}

	// Should have 3 real params (comments filtered out)
	if len(params) != 3 {
		t.Fatalf("params length: got %d, want 3", len(params))
	}

	// Verify param contents
	paramNames := make(map[string]cfgldr.APIParamV1)
	for _, param := range params {
		paramNames[param.NameSpec] = param
	}

	if param, exists := paramNames["q"]; !exists {
		t.Error("missing param 'q'")
	} else {
		if param.Type != "string" {
			t.Errorf("param q type: got %q, want %q", param.Type, "string")
		}
		if param.Constraints != "" {
			t.Errorf("param q constraints: got %q, want empty", param.Constraints)
		}
	}

	if param, exists := paramNames["sort"]; !exists {
		t.Error("missing param 'sort'")
	} else {
		if param.Type != "string" {
			t.Errorf("param sort type: got %q, want %q", param.Type, "string")
		}
		if param.Constraints != "enum[asc,desc]" {
			t.Errorf("param sort constraints: got %q, want %q", param.Constraints, "enum[asc,desc]")
		}
	}

	if param, exists := paramNames["limit"]; !exists {
		t.Error("missing param 'limit'")
	} else {
		if param.Type != "int" {
			t.Errorf("param limit type: got %q, want %q", param.Type, "int")
		}
		if param.Constraints != "range[1..50]" {
			t.Errorf("param limit constraints: got %q, want %q", param.Constraints, "range[1..50]")
		}
	}

	// Verify comment was filtered out
	for _, param := range params {
		if param.NameSpec == "@note" {
			t.Error("comment '@note' should be filtered out from params")
		}
	}
}

func TestAPIEndpointV2_UnmarshalJSON_EmptyParams(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		expectNil bool
	}{
		{
			name: "null params",
			jsonData: `{
				"method": "GET",
				"path": "/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'",
				"params": null
			}`,
			expectNil: false, // should be empty array
		},
		{
			name: "empty array params",
			jsonData: `{
				"method": "GET",
				"path": "/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'",
				"params": []
			}`,
			expectNil: false,
		},
		{
			name: "empty object params",
			jsonData: `{
				"method": "GET",
				"path": "/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'",
				"params": {}
			}`,
			expectNil: false,
		},
		{
			name: "missing params field",
			jsonData: `{
				"method": "GET",
				"path": "/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'"
			}`,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var endpoint cfgldr.APIEndpointV2
			err := jsonv2.Unmarshal([]byte(tt.jsonData), &endpoint)
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if endpoint.Params == nil {
				if !tt.expectNil {
					t.Error("params should not be nil")
				}
			} else {
				params, ok := endpoint.Params.(cfgldr.APIParamsV1)
				if !ok {
					t.Fatalf("params should be APIParamsV1, got %T", endpoint.Params)
				}
				if len(params) != 0 {
					t.Errorf("params should be empty, got %d items", len(params))
				}
			}
		})
	}
}

func TestAPIEndpointV2_NewAPIEndpointV2(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		args     cfgldr.APIEndpointV2Args
		check    func(t *testing.T, ep *cfgldr.APIEndpointV2)
	}{
		{
			name:     "basic endpoint",
			endpoint: "GET /test",
			args: cfgldr.APIEndpointV2Args{
				Description: "Test endpoint",
				Query:       "SELECT 1",
				Cardinality: "one",
				RowType:     "string",
			},
			check: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.Endpoint() != "GET /test" {
					t.Errorf("endpoint: got %q, want %q", ep.Endpoint(), "GET /test")
				}
				if ep.Description != "Test endpoint" {
					t.Errorf("description: got %q, want %q", ep.Description, "Test endpoint")
				}
				if ep.Query != "SELECT 1" {
					t.Errorf("query: got %q, want %q", ep.Query, "SELECT 1")
				}
				if _, ok := ep.Params.(cfgldr.APIParamsV1); !ok {
					t.Errorf("params should default to APIParamsV1, got %T", ep.Params)
				}
				if len(ep.ColumnTypes) != 0 {
					t.Errorf("column_types should default to empty slice, got %v", ep.ColumnTypes)
				}
			},
		},
		{
			name:     "with custom params",
			endpoint: "GET /users/{id:int}",
			args: cfgldr.APIEndpointV2Args{
				Description: "Get user",
				Query:       "SELECT * FROM users WHERE id = :id",
				Params: cfgldr.APIParamsV1{
					{NameSpec: "id", Type: "int", Constraints: ""},
				},
				ColumnTypes: []string{"integer", "string"},
			},
			check: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				params, ok := ep.Params.(cfgldr.APIParamsV1)
				if !ok {
					t.Fatalf("params should be APIParamsV1, got %T", ep.Params)
				}
				if len(params) != 1 {
					t.Fatalf("params length: got %d, want 1", len(params))
				}
				if params[0].NameSpec != "id" {
					t.Errorf("param name: got %q, want %q", params[0].NameSpec, "id")
				}
				if len(ep.ColumnTypes) != 2 {
					t.Errorf("column_types length: got %d, want 2", len(ep.ColumnTypes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, path, _ := strings.Cut(tt.endpoint, " ")
			if path == "" {
				method, path = "", method
			}
			endpoint := cfgldr.NewAPIEndpointV2(method, path, tt.args)
			if endpoint == nil {
				t.Fatal("NewAPIEndpointV2 returned nil")
			}
			tt.check(t, endpoint)
		})
	}
}

func TestAPIEndpointV2_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *cfgldr.APIEndpointV2
		check    func(t *testing.T, ep *cfgldr.APIEndpointV2)
	}{
		{
			name: "empty description gets endpoint value",
			endpoint: &cfgldr.APIEndpointV2{
				APIEndpointBase: cfgldr.APIEndpointBase{
					Method:      "GET",
					Path:        "/test",
					Description: "",
				},
			},
			check: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.Description != "GET /test" {
					t.Errorf("description should default to endpoint, got %q", ep.Description)
				}
			},
		},
		{
			name: "empty cardinality gets default",
			endpoint: &cfgldr.APIEndpointV2{
				APIEndpointBase: cfgldr.APIEndpointBase{
					Method:      "GET",
					Path:        "/test",
					Cardinality: "",
				},
			},
			check: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.Cardinality == "" {
					t.Error("cardinality should be set to default")
				}
			},
		},
		{
			name: "empty row_type gets default",
			endpoint: &cfgldr.APIEndpointV2{
				APIEndpointBase: cfgldr.APIEndpointBase{
					Method:  "GET",
					Path:    "/test",
					RowType: "",
				},
			},
			check: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.RowType == "" {
					t.Error("row_type should be set to default")
				}
			},
		},
		{
			name: "nil params gets empty array",
			endpoint: &cfgldr.APIEndpointV2{
				APIEndpointBase: cfgldr.APIEndpointBase{
					Method: "GET",
					Path:   "/test",
				},
				Params: nil,
			},
			check: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.Params == nil {
					t.Error("params should not be nil after normalize")
				}
				if _, ok := ep.Params.(cfgldr.APIParamsV1); !ok {
					t.Errorf("params should be APIParamsV1, got %T", ep.Params)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.endpoint.Normalize(cfgstore.NormalizeArgs{})
			if err != nil {
				t.Errorf("enpoint failed to normalize: %v", err)
			}
			tt.check(t, tt.endpoint)
		})
	}
}

func TestAPIEndpointV2_UnmarshalJSON_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name: "invalid JSON",
			jsonData: `{
				"method": "GET",
				"path": "/test",
				"params": {invalid json}
			}`,
			wantErr: true,
		},
		{
			name: "invalid param type in map",
			jsonData: `{
				"method": "GET",
				"path": "/test",
				"params": {
					"id": "invalid_type:constraint"
				}
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var endpoint cfgldr.APIEndpointV2
			err := jsonv2.Unmarshal([]byte(tt.jsonData), &endpoint)
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAPIEndpointV2_RealWorldExamples(t *testing.T) {
	// Test with the actual endpoint examples provided by the user
	tests := []struct {
		name     string
		jsonData string
		checks   func(t *testing.T, ep *cfgldr.APIEndpointV2)
	}{
		{
			name: "search tasks endpoint",
			jsonData: `{
				"method": "GET",
				"path": "/tasks/search/{project_id:int}",
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
			}`,
			checks: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.Cardinality != "many" {
					t.Errorf("cardinality: got %q, want %q", ep.Cardinality, "many")
				}
				if ep.RowType != "columns" {
					t.Errorf("row_type: got %q, want %q", ep.RowType, "columns")
				}
				if len(ep.ColumnTypes) != 5 {
					t.Errorf("column_types length: got %d, want 5", len(ep.ColumnTypes))
				}

			},
		},
		{
			name: "tasks by project endpoint",
			jsonData: `{
				"method":"GET",
				"path":"/tasks/by-project/{project:string}",
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
			}`,
			checks: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if len(ep.ColumnTypes) != 8 {
					t.Errorf("column_types length: got %d, want 8", len(ep.ColumnTypes))
				}

				params, ok := ep.Params.(cfgldr.APIParamsV1)
				if !ok {
					t.Fatalf("params should be APIParamsV1, got %T", ep.Params)
				}
				if len(params) != 1 {
					t.Errorf("params length: got %d, want 1", len(params))
				}
				if len(params) > 0 && params[0].NameSpec != "email" {
					t.Errorf("param name: got %q, want %q", params[0].NameSpec, "email")
				}
			},
		},
		{
			name: "simple get user endpoint",
			jsonData: `{
				"method":"GET",
				"path":"/users/{id:int}",
				"description": "Get a single user by numeric id (path parameter only)",
				"query": "SELECT id, email, name, created_at FROM users WHERE id = :id;",
				"cardinality": "one",
				"row_type": "columns",
				"column_types": ["integer", "string", "string", "string"]
			}`,
			checks: func(t *testing.T, ep *cfgldr.APIEndpointV2) {
				if ep.Cardinality != "one" {
					t.Errorf("cardinality: got %q, want %q", ep.Cardinality, "one")
				}

				// Should have empty params (defaults to APIParamsV1{})
				params, ok := ep.Params.(cfgldr.APIParamsV1)
				if !ok {
					t.Fatalf("params should be APIParamsV1, got %T", ep.Params)
				}
				if len(params) != 0 {
					t.Errorf("params should be empty, got %d", len(params))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var endpoint cfgldr.APIEndpointV2
			err := jsonv2.Unmarshal([]byte(tt.jsonData), &endpoint)
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if tt.checks != nil {
				tt.checks(t, &endpoint)
			}
		})
	}
}

func TestAPIEndpointV2_Roundtrip_ArrayFormat(t *testing.T) {
	originalJSON := `{
		"method":"GET",
		"path":"/users/{id:int}",
		"description": "Get a single user by ID",
		"query": "SELECT id, email, name FROM users WHERE id = :id",
		"params": [
			{"name": "id", "type": "int", "constraints": ""},
			{"name": "limit", "type": "int", "constraints": "range[1..100]"}
		],
		"cardinality": "one",
		"row_type": "columns",
		"column_types": ["integer", "string", "string"]
	}`

	// Step 1: Unmarshal original JSON
	var endpoint1 cfgldr.APIEndpointV2
	err := jsonv2.Unmarshal([]byte(originalJSON), &endpoint1)
	if err != nil {
		t.Fatalf("First unmarshal failed: %v", err)
	}

	// Verify it's recognized as array format
	if endpoint1.IsMapFormat() {
		t.Error("endpoint should be recognized as array format, got map format")
	}

	// Step 2: Marshal back to JSON
	marshaledJSON, err := endpoint1.MarshalJSON()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Step 3: Unmarshal the marshaled JSON
	var endpoint2 cfgldr.APIEndpointV2
	err = jsonv2.Unmarshal(marshaledJSON, &endpoint2)
	if err != nil {
		t.Fatalf("Second unmarshal failed: %v", err)
	}

	// Step 4: Verify data integrity
	if endpoint2.Endpoint() != endpoint1.Endpoint() {
		t.Errorf("endpoint mismatch: got %q, want %q", endpoint2.Endpoint(), endpoint1.Endpoint())
	}
	if endpoint2.Description != endpoint1.Description {
		t.Errorf("description mismatch: got %q, want %q", endpoint2.Description, endpoint1.Description)
	}

	// Verify params are identical
	params1, ok1 := endpoint1.Params.(cfgldr.APIParamsV1)
	params2, ok2 := endpoint2.Params.(cfgldr.APIParamsV1)
	if !ok1 || !ok2 {
		t.Fatalf("params should be APIParamsV1, got %T and %T", endpoint1.Params, endpoint2.Params)
	}
	if len(params1) != len(params2) {
		t.Fatalf("params length mismatch: got %d, want %d", len(params2), len(params1))
	}
	for i, p1 := range params1 {
		p2 := params2[i]
		if p1.NameSpec != p2.NameSpec || p1.Type != p2.Type || p1.Constraints != p2.Constraints {
			t.Errorf("param %d mismatch: got %+v, want %+v", i, p2, p1)
		}
	}

	// Verify format is preserved
	if endpoint2.IsMapFormat() {
		t.Error("format should remain array after roundtrip")
	}
}

func TestAPIEndpointV2_Roundtrip_MapFormat(t *testing.T) {
	originalJSON := `{
		"method":"GET",
			"path":"/tasks/search/{project_id:int}",
		"description": "Search tasks within a given project",
		"query": "SELECT t.id, t.title FROM tasks t WHERE t.project_id = :project_id",
		"params": {
			"q": "string",
			"sort": "string:enum[asc,desc]",
			"limit": "int:range[1..50]",
			"@note": ["Just a little bit of info", "for posterity"]
		},
		"cardinality": "many",
		"row_type": "columns",
		"column_types": ["integer", "string"]
	}`

	// Step 1: Unmarshal original JSON
	var endpoint1 cfgldr.APIEndpointV2
	err := jsonv2.Unmarshal([]byte(originalJSON), &endpoint1)
	if err != nil {
		t.Fatalf("First unmarshal failed: %v", err)
	}

	// Verify it's recognized as map format
	if !endpoint1.IsMapFormat() {
		t.Error("endpoint should be recognized as map format, got array format")
	}

	// Step 2: Marshal back to JSON
	marshaledJSON, err := endpoint1.MarshalJSON()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Step 3: Unmarshal the marshaled JSON
	var endpoint2 cfgldr.APIEndpointV2
	err = jsonv2.Unmarshal(marshaledJSON, &endpoint2)
	if err != nil {
		t.Fatalf("Second unmarshal failed: %v", err)
	}

	// Step 4: Verify data integrity
	if endpoint2.Endpoint() != endpoint1.Endpoint() {
		t.Errorf("endpoint mismatch: got %q, want %q", endpoint2.Endpoint(), endpoint1.Endpoint())
	}
	if endpoint2.Description != endpoint1.Description {
		t.Errorf("description mismatch: got %q, want %q", endpoint2.Description, endpoint1.Description)
	}

	// Verify params contain the same data (though internally stored as APIParamsV1)
	params1, ok1 := endpoint1.Params.(cfgldr.APIParamsV1)
	params2, ok2 := endpoint2.Params.(cfgldr.APIParamsV1)
	if !ok1 || !ok2 {
		t.Fatalf("params should be APIParamsV1, got %T and %T", endpoint1.Params, endpoint2.Params)
	}
	if len(params1) != len(params2) {
		t.Fatalf("params length mismatch: got %d, want %d", len(params2), len(params1))
	}

	// Convert to maps for easier comparison
	params1Map := params1.APIParamsMap()
	params2Map := params2.APIParamsMap()

	// Check that all non-comment keys are preserved
	for key, value := range params1Map.Iterator() {
		if len(key) > 0 && key[0] == '@' {
			continue // Skip comments
		}
		value2, exists := params2Map.Get(key)
		if !exists {
			t.Errorf("missing key %q in roundtrip result", key)
		} else if value != value2 {
			t.Errorf("value mismatch for key %q: got %q, want %q", key, value2, value)
		}
	}

	// Verify format is preserved
	if !endpoint2.IsMapFormat() {
		t.Error("format should remain map after roundtrip")
	}
}

func TestAPIEndpointV2_Roundtrip_EmptyParams(t *testing.T) {
	tests := []struct {
		name         string
		originalJSON string
		expectMap    bool
	}{
		{
			name: "empty array",
			originalJSON: `{
				"method":"GET",
			"path":"/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'",
				"params": []
			}`,
			expectMap: false,
		},
		{
			name: "empty object",
			originalJSON: `{
				"method":"GET",
				"path":"/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'",
				"params": {}
			}`,
			expectMap: true,
		},
		{
			name: "null params",
			originalJSON: `{
				"method":"GET",
				"path":"/hello",
				"description": "Hello World",
				"query": "SELECT 'Hello World'",
				"params": null
			}`,
			expectMap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Unmarshal original JSON
			var endpoint1 cfgldr.APIEndpointV2
			err := jsonv2.Unmarshal([]byte(tt.originalJSON), &endpoint1)
			if err != nil {
				t.Fatalf("First unmarshal failed: %v", err)
			}

			// Verify format detection
			if endpoint1.IsMapFormat() != tt.expectMap {
				t.Errorf("format detection: got %v, want %v", endpoint1.IsMapFormat(), tt.expectMap)
			}

			// Step 2: Marshal back to JSON
			marshaledJSON, err := endpoint1.MarshalJSON()
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Step 3: Unmarshal the marshaled JSON
			var endpoint2 cfgldr.APIEndpointV2
			err = jsonv2.Unmarshal(marshaledJSON, &endpoint2)
			if err != nil {
				t.Fatalf("Second unmarshal failed: %v", err)
			}

			// Step 4: Verify format is preserved
			if endpoint2.IsMapFormat() != tt.expectMap {
				t.Errorf("format preservation: got %v, want %v", endpoint2.IsMapFormat(), tt.expectMap)
			}

			// Verify params are empty
			params, ok := endpoint2.Params.(cfgldr.APIParamsV1)
			if !ok {
				t.Fatalf("params should be APIParamsV1, got %T", endpoint2.Params)
			}
			if len(params) != 0 {
				t.Errorf("params should be empty, got %d items", len(params))
			}
		})
	}
}
