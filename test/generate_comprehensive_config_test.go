package test

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"os"

	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

// generateConfig generates the api__test.json configuration file.
// This function is called from TestMain() to ensure the config exists before tests run.
// It can also be called manually via TestGenerateConfig for regeneration.
func generateConfig(outputPath string) error {
	api := cfgldr.NewAPIConfigV2("./.xmlui")
	api.Name = " PathVars Integration Test API"

	// Helper function to create params map
	m := func(pairs ...string) *cfgldr.APIParamsMap {
		pm := &cfgldr.APIParamsMap{}
		pm.Clear() // Initialize internal structures
		for i := 0; i < len(pairs); i += 2 {
			pm.Set(cfgldr.APIParamsMapKey(pairs[i]), cfgldr.APIParamsMapValue(pairs[i+1]))
		}
		return pm
	}

	// Helper function for array params
	arrayParams := func(params ...cfgldr.APIParamV1) cfgldr.APIParamsV1 {
		return params
	}

	param := func(name, typ, constraints string) cfgldr.APIParamV1 {
		return cfgldr.APIParamV1{
			NameSpec:    name,
			Type:        typ,
			Constraints: constraints,
		}
	}

	// Add all 27 endpoints from the  test
	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/tasks/by-estimate/{estimate:real}?{min_priority?1:int:range[1..5]}&{status?todo:string:enum[todo,doing,done]}", cfgldr.APIEndpointV2Args{
		Description: "Real path parameter with optional query parameters with defaults",
		Query:       "SELECT id, title, status, priority, estimate FROM tasks WHERE estimate >=:estimate AND priority >=:min_priority AND status = :status;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer", "real"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-score/{score:int:range[0..100]}", cfgldr.APIEndpointV2Args{
		Description: "Integer range constraint",
		Query:       "SELECT id, email, name, score FROM users WHERE score = :score;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/logs/by-path/{file_path*}", cfgldr.APIEndpointV2Args{
		Description: "Multi-segment path parameter for file paths",
		Query:       "SELECT id, level, message, file_path, line_number FROM logs WHERE file_path LIKE :file_path || '%';",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string", "integer"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/hello", cfgldr.APIEndpointV2Args{
		Description: "Simple endpoint with no parameters",
		Query:       "SELECT 'Hello World' as message;",
		Params:      arrayParams(),
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"string"},
	}))

	// Add /users/search BEFORE /users/{id:int} to ensure proper route matching
	// (literal paths should be checked before parameterized paths)
	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/search?{email:string:notempty}&{active?true:boolean}&{min_score?0:int}&{limit?10:int:range[1..100]}", cfgldr.APIEndpointV2Args{
		Description: "Query-only endpoint with notempty constraint and defaults",
		Query:       "SELECT id, email, name, score, active FROM users WHERE email LIKE '%' ||:email || '%' AND active =:active AND score >=:min_score ORDER BY score DESC LIMIT :limit;",
		Params: m(
			"@note", "email uses generic 'string' type (not 'email' type) to allow partial matching with LIKE query.\nTests explicit type declaration vs implicit type inference.\nactive parameter is boolean but maps to INTEGER column (SQLite stores booleans as 0/1).\nTests Database.ConvertValue() interface for type compatibility.",
		),
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer", "integer"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/{id:int}", cfgldr.APIEndpointV2Args{
		Description: "Basic integer path parameter",
		Query:       "SELECT id, email, name, slug FROM users WHERE id = :id;",
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-uuid/{uuid:uuid:format[v4]}", cfgldr.APIEndpointV2Args{
		Description: "UUID v4 format constraint",
		Query:       "SELECT id, email, name FROM users WHERE uuid = :uuid;",
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-slug/{slug:slug:length[5..50]}", cfgldr.APIEndpointV2Args{
		Description: "Slug parameter with length constraint",
		Query:       "SELECT id, email, name FROM users WHERE slug = :slug;",
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-rating/{rating:real:range[0.0..5.0]}", cfgldr.APIEndpointV2Args{
		Description: "Real number with decimal range constraint",
		Query:       "SELECT id, email, name, rating FROM users WHERE rating >= :rating;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "real"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-status/{active:boolean}", cfgldr.APIEndpointV2Args{
		Description: "Boolean parameter",
		Query:       "SELECT id, email, name, active FROM users WHERE active = :active;",
		Params: m(
			"@note", "Boolean path parameter maps to INTEGER column in SQLite.\nTests automatic conversion via Database.ConvertValue() (true→1, false→0).",
		),
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-birth-date/{birth_date:date:format[yyyy-mm-dd]}", cfgldr.APIEndpointV2Args{
		Description: "Date parameter with ISO format",
		Query:       "SELECT id, email, name, birth_date FROM users WHERE birth_date = :birth_date;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-login-time/{login_time:date:format[hh:mm:ss]}", cfgldr.APIEndpointV2Args{
		Description: "Time parameter",
		Query:       "SELECT id, email, name, login_time FROM users WHERE login_time = :login_time;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/users/by-name/{name:string}", cfgldr.APIEndpointV2Args{
		Description: "Name parameter",
		Query:       "SELECT id, email, name FROM users WHERE name = :name;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/implicit/{int}", cfgldr.APIEndpointV2Args{
		Description: "Implicit type inference for int",
		Query:       "SELECT id, email, name FROM users WHERE id = :int;",
		Params: m(
			"@note", "Tests implicit type inference: parameter name 'int' is recognized as integer type.\nNo explicit :type declaration needed.",
		),
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/implicit/{slug}", cfgldr.APIEndpointV2Args{
		Description: "Implicit type inference for slug",
		Query:       "SELECT id, email, name FROM users WHERE slug = :slug;",
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/implicit/{uuid}", cfgldr.APIEndpointV2Args{
		Description: "Implicit type inference for uuid",
		Query:       "SELECT id, email, name FROM users WHERE uuid = :uuid;",
		Cardinality: string(sqlparams.OneRow),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/implicit/{decimal::range[0.0..1000.0]}", cfgldr.APIEndpointV2Args{
		Description: "Implicit decimal type with double colon constraint syntax",
		Query:       "SELECT sensor_id, value, unit FROM measurements WHERE precision_val = :decimal;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"string", "real", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/projects/by-status/{status:string:enum[active,archived,draft]}", cfgldr.APIEndpointV2Args{
		Description: "String with enum constraint",
		Query:       "SELECT id, name, status, priority FROM projects WHERE status = :status;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/projects/by-priority/{priority:int:range[1..5]}", cfgldr.APIEndpointV2Args{
		Description: "Integer range constraint for priority",
		Query:       "SELECT id, name, status, priority, budget FROM projects WHERE priority = :priority;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer", "real"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/projects/by-budget/{budget:decimal:range[1000.0..50000.0]}", cfgldr.APIEndpointV2Args{
		Description: "Decimal parameter with range constraint",
		Query:       "SELECT id, name, budget FROM projects WHERE budget >= :budget;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "real"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/tasks/search/{project_id:int}", cfgldr.APIEndpointV2Args{
		Description: "Path parameter with query parameters",
		Query:       "SELECT t.id, t.title, t.status, t.priority, IFNULL(u.email,'') AS assignee_email FROM tasks t LEFT JOIN users u ON u.id = t.assignee_id WHERE t.project_id =:project_id AND (LOWER(t.title) LIKE LOWER('%' ||:q || '%') OR LOWER(t.details) LIKE LOWER('%' ||:q || '%')) ORDER BY t.priority DESC LIMIT :limit OFFSET :offset;",
		Params: m(
			"q", "string",
			"limit", "int:range[1..100]",
			"offset", "int:range[0..1000]",
			"sort", "string:enum[asc,desc]",
			"@note", "Query parameter testing\nwith path parameter combination",
		),
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "integer", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/logs/by-level/{level:string:enum[DEBUG,INFO,WARN,ERROR]}", cfgldr.APIEndpointV2Args{
		Description: "String enum for log levels",
		Query:       "SELECT id, level, message, file_path FROM logs WHERE level = :level;",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(sqlparams.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/archive/by-date/{archive_date*:date:format[yyyy/mm/dd]}", cfgldr.APIEndpointV2Args{
		Description: "Multi-segment date parameter with slash format",
		Query:       "SELECT id, archive_date, content, format_type FROM archive WHERE archive_date LIKE :archive_date || '%';",
		Cardinality: string(sqlparams.ManyRows),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/archive/by-datetime/{datetime:date:format[yyyy-mm-dd_hh:mm:ss]}", cfgldr.APIEndpointV2Args{
		Description: "Combined date-time format",
		Query:       "SELECT id, archive_date, archive_time, content FROM archive WHERE archive_date || '_' || archive_time = :datetime;",
		Cardinality: string(dbqvars.ManyRows),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/measurements/by-sensor/{sensor_id:alphanumeric:length[4..10]}", cfgldr.APIEndpointV2Args{
		Description: "Alphanumeric parameter with length constraint",
		Query:       "SELECT sensor_id, value, unit, precision_val FROM measurements WHERE sensor_id = :sensor_id;",
		Cardinality: string(dbqvars.ManyRows),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"string", "real", "string", "real"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/measurements/by-value/{value:real:range[0.0..1000.5]}", cfgldr.APIEndpointV2Args{
		Description: "Real parameter with decimal range constraint",
		Query:       "SELECT sensor_id, value, unit FROM measurements WHERE value = :value;",
		Cardinality: string(dbqvars.ManyRows),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"string", "real", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/projects/search?{name?:string}&{status?active:string:enum[active,archived,draft]}&{min_budget?0:decimal}", cfgldr.APIEndpointV2Args{
		Description: "All optional query parameters",
		Query:       "SELECT id, name, status, budget FROM projects WHERE (:name = '' OR name LIKE '%' || :name || '%') AND status = :status AND budget >= :min_budget;",
		Params: m(
			"@note", "All parameters optional with defaults: name defaults to empty string, status to 'active', min_budget to 0.\nTests optional parameter handling and default value application.",
		),
		Cardinality: string(dbqvars.ManyRows),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string", "real"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("POST", "/users/{id:int}/update", cfgldr.APIEndpointV2Args{
		Description: "POST endpoint with path parameter and JSON body",
		Query:       "UPDATE users SET name = :name, email = :email WHERE id = :id; SELECT id, name, email FROM users WHERE id = :id;",
		Params: arrayParams(
			param("name", "string", "length[1..100]"),
			param("email", "string", "regex[[^@]+@[^@]+\\.[^@]+]"),
		),
		Cardinality: string(dbqvars.OneRow),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"integer", "string", "string"},
	}))

	api.AddEndpoint(cfgldr.NewAPIEndpointV2("PUT", "/projects/{project_id:int}/tasks", cfgldr.APIEndpointV2Args{
		Description: "PUT endpoint with nested JSON body parameters",
		Query:       "INSERT INTO tasks (project_id, title, details, status, priority, estimate) VALUES (:project_id, :task.title, :task.details, :task.status, :task.priority, :task.estimate); SELECT last_insert_rowid() as id;",
		Params: arrayParams(
			param("task.title", "string", "length[1..200]"),
			param("task.details", "string", ""),
			param("task.status", "string", "enum[todo,doing,done]"),
			param("task.priority", "int", "range[1..5]"),
			param("task.estimate", "real", "range[0.0..100.0]"),
		),
		Cardinality: string(dbqvars.OneRow),
		RowType:     string(dbqvars.ColumnsRowType),
		ColumnTypes: []string{"integer"},
	}))

	// Create database config
	db := cfgldr.NewSQLite3ConfigV1("test.db")
	db.OnOpenSQL = []string{"PRAGMA foreign_keys = OFF;"}

	// Create server config
	server := cfgldr.NewServerConfigV1(common.DefaultServerHost, cfgldr.ServerConfigV1Args{
		Port: 8080,
		API:  api,
		Notes: []string{
			"GENERATED FILE: DO NOT EDIT!!!",
			"Edit ./test/generate__config_test.go instead.",
		},
	})

	// Create root config
	root := cfgldr.NewRootConfigV1(cfgldr.RootConfigV1Args{
		ServerConfig: server,
		DBConfig:     db,
	})

	// Write to file
	bytes, err := jsonv2.Marshal(root, jsontext.WithIndent("   "))
	if err != nil {
		return err
	}

	err = os.WriteFile(outputPath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
