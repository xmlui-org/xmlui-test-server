package cfgldr

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"os"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	RootConfigV1Version = 1
	RootConfigV1Schema  = "https://xmlui.org/schemas/v1/localsvr/root-schema.json"
)

var _ Config = (*RootConfigV1)(nil)
var _ cfgstore.RootConfig = (*RootConfigV1)(nil)

// RootConfigV1 represents the root configuration structure as defined in ADR-001
type RootConfigV1 struct {
	rootConfigV1Base `json:",inline"`
	DBConfig         DatabaseConfig `json:"database"`
}

func (c *RootConfigV1) Merge(rc cfgstore.RootConfig) cfgstore.RootConfig {
	// Merge rc (base config) into c (receiver)
	// c takes precedence - only fill in empty/nil fields from rc
	var base *RootConfigV1
	switch t := rc.(type) {
	case *RootConfigV1:
		base = t
	case *RootConfigV1Wrapper:
		base = &t.RootConfigV1
	default:
		return c
	}

	// Merge ServerConfig
	if c.ServerConfig == nil {
		c.ServerConfig = base.ServerConfig
	} else if base.ServerConfig != nil {
		c.ServerConfig = c.ServerConfig.Merge(base.ServerConfig)
	}

	// Merge DBConfig
	if c.DBConfig == nil {
		c.DBConfig = base.DBConfig
	} else if base.DBConfig != nil {
		c.DBConfig = c.DBConfig.Merge(base.DBConfig)
	}

	return c
}

func (c *RootConfigV1) RootConfig() {}

// Base struct with non-polymorphic fields
type rootConfigV1Base struct {
	Schema       string           `json:"$schema"`
	Version      int              `json:"version"`
	Project      string           `json:"project"`
	ServerConfig *ServerConfigV1  `json:"server"`
	DirType      cfgstore.DirType `json:"-"`
}

type RootConfigV1Args struct {
	Project      string `json:"project"`
	ServerConfig *ServerConfigV1
	DBConfig     DatabaseConfig
}

func NewRootConfigV1(args RootConfigV1Args) *RootConfigV1 {
	return &RootConfigV1{
		rootConfigV1Base: rootConfigV1Base{
			Schema:       RootConfigV1Schema,
			Version:      RootConfigV1Version,
			Project:      args.Project,
			ServerConfig: args.ServerConfig,
		},
		DBConfig: args.DBConfig,
	}
}

func (c *RootConfigV1) APIConfig() (ac APIConfig) {
	if c == nil {
		goto end
	}
	if c.ServerConfig == nil {
		goto end
	}
	ac = c.ServerConfig.APIConfig
end:
	return ac
}

func (c *RootConfigV1) Config() {}

func (c *RootConfigV1) Normalize(args cfgstore.NormalizeArgs) error {
	var errs []error
	c.Schema = RootConfigV1Schema
	c.Version = RootConfigV1Version
	c.DirType = args.DirType
	if c.ServerConfig == nil {
		c.ServerConfig = NewServerConfigV1(localsvr.DefaultServerHost, ServerConfigV1Args{
			Port: localsvr.DefaultServerPort,
			API:  NewAPIConfigV2(localsvr.DefaultWebroot),
		})
	}
	errs = AppendErr(errs, c.ServerConfig.Normalize(args))
	if c.DBConfig == nil {
		c.DBConfig = NewSQLite3ConfigV1(localsvr.DefaultSQLite3Database)
	}
	errs = AppendErr(errs, c.DBConfig.Normalize(args))
	return CombineErrs(errs)
}

func (c *RootConfigV1) String() string {
	return string(c.Bytes())
}

func (c *RootConfigV1) Bytes() []byte {
	b, err := jsonv2.Marshal(c, jsontext.WithIndent("  "))
	if err != nil {
		panic(err)
	}
	return b
}

func (c *RootConfigV1) UnmarshalJSON(data []byte) (err error) {
	var dbc DatabaseConfig

	// Create temp struct with inline base and RawMessage for polymorphic field
	var temp struct {
		rootConfigV1Base `json:",inline"`
		Database         jsontext.Value `json:"database"`
	}

	var typeInfo struct {
		Type DatabaseType `json:"type"`
	}

	err = jsonv2.Unmarshal(data, &temp)
	if err != nil {
		goto end
	}

	// Copy non-polymorphic fields from temp
	c.rootConfigV1Base = temp.rootConfigV1Base

	// Handle polymorphic database field (skip if not present)
	if len(temp.Database) == 0 {
		goto end
	}

	err = jsonv2.Unmarshal(temp.Database, &typeInfo)
	if err != nil {
		goto end
	}

	dbc, err = GetDatabaseConfig(typeInfo.Type)
	if err != nil {
		goto end
	}

	dbc = dbc.Clone()

	err = jsonv2.Unmarshal(temp.Database, &dbc)
	if err != nil {
		goto end
	}

	c.DBConfig = dbc

end:
	return err
}

type LoadRootConfigV1Args struct {
	AppInfo      appinfo.AppInfo
	Options      cfgstore.Options
	DirTypes     []cfgstore.DirType
	ConfigStores *cfgstore.ConfigStores
	DirsProvider *cfgstore.DirsProvider
}

var _ cfgstore.RootConfig = (*RootConfigV1Wrapper)(nil)

type RootConfigV1Wrapper struct {
	RootConfigV1
}

func (w *RootConfigV1Wrapper) SetRootConfigV1(c RootConfigV1) {
	w.RootConfigV1 = c
}

func (w *RootConfigV1Wrapper) Normalize(args cfgstore.NormalizeArgs) (err error) {
	var co *Options
	co, err = dtx.AssertType[*Options](args.Options)
	if err != nil {
		goto end
	}
	err = w.RootConfigV1.Normalize(cfgstore.NormalizeArgs{
		DirType:    args.DirType,
		SourceFile: args.SourceFile,
		Options:    co,
	})
end:
	return err
}
func (w *RootConfigV1Wrapper) MarshalJSON() ([]byte, error) {
	return jsonv2.Marshal(w.RootConfigV1)
}

func (w *RootConfigV1Wrapper) UnmarshalJSON(b []byte) error {
	return jsonv2.Unmarshal(b, &w.RootConfigV1)
}

func LoadRootConfigV1(args LoadRootConfigV1Args) (_ *RootConfigV1, err error) {
	var lrc *RootConfigV1Wrapper
	var rc RootConfigV1
	var opts *Options
	var bootstrapBytes []byte

	configStores := args.ConfigStores
	if configStores == nil {
		configStores = cfgstore.NewConfigStores(cfgstore.ConfigStoresArgs{
			ConfigStoreArgs: cfgstore.ConfigStoreArgs{
				ConfigSlug:   args.AppInfo.ConfigSlug(),
				RelFilepath:  args.AppInfo.ConfigFile(),
				DirsProvider: args.DirsProvider,
			},
		})
	}

	// Get externally set options such as via the switches on the command line
	lrc, err = cfgstore.LoadConfigStores[RootConfigV1Wrapper, *RootConfigV1Wrapper](configStores, cfgstore.RootConfigArgs{
		DirTypes:     args.DirTypes,
		Options:      args.Options,
		DirsProvider: args.DirsProvider,
	})
	if err != nil {
		goto end
	}
	if lrc == nil {
		panic("LoadConfigStores() returned nil")
	}
	rc = lrc.RootConfigV1

	// Load bootstrap SQL file if specified
	opts, err = dtx.AssertType[*Options](args.Options)
	if err != nil {
		goto end
	}
	if opts.DBBootstrapFile != "" {
		bootstrapBytes, err = dt.Filepath(opts.DBBootstrapFile).ReadFile()
		if err != nil && !os.IsNotExist(err) {
			err = WithErr(err,
				ErrFailedToLoadDBSchemaFile,
				"dbschema_file", opts.DBBootstrapFile,
			)
			goto end
		}
		if len(bootstrapBytes) != 0 {
			rc.DBConfig.SetBootstrapQueries([]string{string(bootstrapBytes)})
		}
		err = nil
	}
end:
	return &rc, err
}

type GenerateConfigArgs struct {
	Project     string   // Name of project
	Webroot     string   // Path to webroot directory (default: ".")
	DBPath      string   // Path to database file (default: "db/data.db")
	DBBootstrap string   // Path to bootstrap SQL file (default: "db/bootstrap.sql")
	Port        int      // HTTP port (default: 8080)
	Host        string   // HTTP host (default: "127.0.0.1")
	OnOpenSQL   []string // SQL statements to run on database open
}

// GenerateConfig creates a minimal, sensible RootConfigV1 with the specified parameters.
// Any zero-value parameters will use defaults from the cfgldr constants.
func GenerateConfig(args GenerateConfigArgs) *RootConfigV1 {
	// Apply defaults
	if args.Project == "" {
		wd, err := dt.Getwd()
		switch err != nil {
		case true:
			args.Project = "Project name goes here"
		default:
			caser := cases.Title(language.Und)
			args.Project = caser.String(string(wd.Base()))
		}
	}
	if args.Webroot == "" {
		args.Webroot = localsvr.DefaultWebroot
	}
	if args.DBPath == "" {
		args.DBPath = localsvr.DefaultSQLite3Database
	}
	if args.DBBootstrap == "" {
		args.DBBootstrap = localsvr.DefaultDBBootstrapFilepath
	}
	if args.Port == 0 {
		args.Port = localsvr.DefaultServerPort
	}
	if args.Host == "" {
		args.Host = localsvr.DefaultServerHost
	}

	// Create minimal API config with just webroot
	api := NewAPIConfigV2(args.Webroot)

	// Create database config
	db := NewSQLite3ConfigV1(args.DBPath)
	db.Extensions = nil
	if args.OnOpenSQL != nil {
		db.OnOpenSQL = args.OnOpenSQL
	}

	// Create server config
	server := NewServerConfigV1(args.Host, ServerConfigV1Args{
		Port: args.Port,
		API:  api,
	})

	// Create and return root config
	return NewRootConfigV1(RootConfigV1Args{
		Project:      args.Project,
		ServerConfig: server,
		DBConfig:     db,
	})
}

//
//func ensureConfig(cs cfgstore.ConfigStore, opts *Options) (rc *RootConfigV1, err error) {
//	rc, err = loadConfigIfExists(cs, opts)
//	if err != nil {
//		// A real error occurred, bail out
//		goto end
//	}
//
//	if rc == nil {
//		// Config not loaded, need to create config
//		rc, err = createConfig(cs, opts)
//		goto end
//	}
//
//end:
//	return rc, err
//}
//
//func createConfig(cs cfgstore.ConfigStore, opts *Options) (rc *RootConfigV1, err error) {
//	var api *APIConfigV2
//	var db *SQLite3ConfigV1
//	var server *ServerConfigV1
//	var fp dt.Filepath
//
//	api = NewAPIConfigV2(localsvr.DefaultWebroot)
//
//	m := &APIParamsMap{}
//	m.Set("q", "string")
//	m.Set("sort", "string:enum[asc,desc]")
//	m.Set("limit", "int:range[1..50]")
//	m.Set("@note", "Just a little bit of info\nfor posterity")
//
//	const TasksSearchByProjectSQL = `SELECT t.id, t.title, t.status, t.priority, IFNULL(au.email,'') AS assignee_email FROM tasks t LEFT JOIN users au ON au.id = t.assignee_id WHERE t.project_id = :project_id AND (LOWER(t.title) LIKE LOWER('%' || :q || '%') OR LOWER(t.details) LIKE LOWER('%' || :q || '%')) ORDER BY t.priority DESC, t.id;`
//
//	// Add tasks search endpoint with path parameter and params map
//	api.AddEndpoint(NewAPIEndpointV2("GET", "/tasks/search/{project_id:int}", APIEndpointV2Args{
//		Description: "Search tasks within a given project (path param project_id + query-string param q)",
//		Query:       TasksSearchByProjectSQL,
//		Cardinality: string(sqlparams.ManyRows),
//		RowType:     string(sqlparams.ColumnsRowType),
//		ColumnTypes: []string{
//			string(sqlparams.IntegerDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.IntegerDBDataType),
//			string(sqlparams.StringDBDataType),
//		},
//		Params: m,
//	}))
//
//	const TasksByProjectSQL = `SELECT t.id, t.title, t.status, t.priority, t.due_date, au.email AS assignee_email, au.name AS assignee_name, t.created_at FROM tasks t JOIN projects p ON p.id = t.project_id JOIN users ou ON ou.id = p.owner_id LEFT JOIN users au ON au.id = t.assignee_id WHERE ou.email = :email AND p.name = :project ORDER BY t.priority DESC, t.created_at;`
//
//	// Add tasks by project endpoint with array params
//	api.AddEndpoint(NewAPIEndpointV2("GET", "/tasks/by-project/{project:string}", APIEndpointV2Args{
//		Description: "Tasks for a project using project in the path and owner email as a query-string parameter",
//		Query:       TasksByProjectSQL,
//		Cardinality: string(sqlparams.ManyRows),
//		RowType:     string(sqlparams.ColumnsRowType),
//		ColumnTypes: []string{
//			string(sqlparams.IntegerDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.IntegerDBDataType),
//			string(sqlparams.StringDBDataTypeOrNULL),
//			string(sqlparams.StringDBDataTypeOrNULL),
//			string(sqlparams.StringDBDataTypeOrNULL),
//			string(sqlparams.StringDBDataType),
//		},
//		Params: APIParamsV1{
//			{NameSpec: "email", Type: "string"},
//		},
//	}))
//
//	var UserByIdSQL = `SELECT id, email, name, created_at FROM users WHERE id = :id;`
//	// Add user by ID endpoint
//	api.AddEndpoint(NewAPIEndpointV2("GET", "/users/{id:int}", APIEndpointV2Args{
//		Description: "Get a single user by numeric id (path parameter only)",
//		Query:       UserByIdSQL,
//		Cardinality: string(sqlparams.OneRow),
//		RowType:     string(sqlparams.ColumnsRowType),
//		ColumnTypes: []string{
//			string(sqlparams.IntegerDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.StringDBDataType),
//		},
//	}))
//
//	// Add hello world endpoint
//	api.AddEndpoint(NewAPIEndpointV2("GET", "/hello", APIEndpointV2Args{
//		Description: "Hello World Endpoint",
//		Query:       "SELECT 'Hello World';",
//		Cardinality: string(sqlparams.OneRow),
//		RowType:     string(sqlparams.StringRowType),
//		Params:      APIParamsV1{},
//	}))
//
//	const ProjectsByOwnersEmailSQL = `SELECT p.id, p.name, p.status, p.created_at FROM projects p WHERE p.owner_id = (SELECT id FROM users WHERE email = :email) ORDER BY p.created_at DESC;`
//	// Add projects by owner endpoint
//	api.AddEndpoint(NewAPIEndpointV2("GET", "/projects/by-owner/{email:string}", APIEndpointV2Args{
//		Description: "Projects owned by a given user (owner email as a path parameter)",
//		Query:       ProjectsByOwnersEmailSQL,
//		Cardinality: string(sqlparams.ManyRows),
//		RowType:     string(sqlparams.ColumnsRowType),
//		ColumnTypes: []string{
//			string(sqlparams.IntegerDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.StringDBDataType),
//			string(sqlparams.StringDBDataType),
//		},
//	}))
//
//	db = NewSQLite3ConfigV1(localsvr.DefaultSQLite3Database)
//	db.Extensions = nil
//	db.OnOpenSQL = []string{"PRAGMA foreign_keys = OFF;"}
//
//	server = NewServerConfigV1(localsvr.DefaultServerHost, ServerConfigV1Args{
//		Port: 8080,
//		API:  api,
//	})
//	rc = NewRootConfigV1(RootConfigV1Args{
//		ServerConfig: server,
//		DBConfig:     db,
//	})
//	fp, err = cs.GetFilepath()
//	if err != nil {
//		goto end
//	}
//	err = rc.Normalize(args)
//	if err != nil {
//		goto end
//	}
//	err = cs.SaveJSON(rc)
//	if err != nil {
//		goto end
//	}
//end:
//	return rc, err
//}
//
//func loadConfigIfExists(cs cfgstore.ConfigStore, opts *Options) (rc *RootConfigV1, err error) {
//	var fp dt.Filepath
//
//	if !cs.Exists() {
//		goto end
//	}
//
//	rc = &RootConfigV1{}
//	err = cs.LoadJSON(rc, nil)
//	if err != nil {
//		goto end
//	}
//	fp, err = cs.GetFilepath()
//	if err != nil {
//		goto end
//	}
//	err = rc.Normalize(args)
//end:
//	return rc, err
//}
//
//func loadAPIFileIfExists(apiFile string) (api *APIConfigV2, err error) {
//	var apiBytes []byte
//	if apiFile == "" {
//		goto end
//	}
//	apiBytes, err = cfgstore.ReadFileIfExists(apiFile)
//	if err != nil {
//		err = NewErr(ErrFailedToLoadAPIConfigFile, err)
//		goto end
//	}
//	api = &APIConfigV2{}
//	err = jsonv2.Unmarshal(apiBytes, &api)
//	if err != nil {
//		err = NewErr(ErrFailedToUnmarshalAPIConfigFile, err)
//		goto end
//	}
//end:
//	if err != nil {
//		err = WithErr(err,
//			"api_file", apiFile,
//		)
//	}
//	return api, err
//}
