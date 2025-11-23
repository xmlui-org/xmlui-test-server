package cfgldr_test

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-jsontest"
	_ "github.com/mikeschinkel/go-jsontest/pipefuncs"
	"github.com/stretchr/testify/require"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

// TestCreateGoldenData is not a real test but a convenience to write a "Golden" file we can cherry pick from
func TestCreateGoldenData(t *testing.T) {
	api := cfgldr.NewAPIConfigV2(".")
	api.AddEndpoint(cfgldr.NewAPIEndpointV2("GET", "/hello", cfgldr.APIEndpointV2Args{
		Description: "Hello World Endpoint",
		Query:       "SELECT 'Hello World';",
		Cardinality: "one",
		RowType:     "string",
	}))
	db := cfgldr.NewSQLite3ConfigV1("data.db")
	ext := &cfgldr.SQLite3ExtensionConfigV1{
		Filepath: "steampipe_sqlite_github.so",
	}
	err := db.AddExtension(ext, nil)
	if err != nil {
		t.Error(err.Error())
	}
	server := cfgldr.NewServerConfigV1(common.LocalHostIP, cfgldr.ServerConfigV1Args{
		Port: 8080,
		API:  api,
	})
	root := cfgldr.NewRootConfigV1(cfgldr.RootConfigV1Args{
		ServerConfig: server,
		DBConfig:     db,
	})
	err = writeJSONFile("./test-data/localsvr.json", root, 0644, 0755)
	if err != nil {
		t.Fatal(err.Error())
	}
}
func TestLoadRootConfigV1(t *testing.T) {
	type args struct {
		appName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name: "LoadJSON Config",
			args: args{},
			want: map[string]any{
				"$schema":                                   "https://xmlui.org/schemas/v1/localsvr/root-schema.json",
				"version":                                   1,
				"server|exists()":                           true,
				"server.$schema":                            "https://xmlui.org/schemas/v1/localsvr/server-schema.json",
				"server.version":                            1,
				"server.host":                               "127.0.0.1",
				"server.port":                               8080,
				"server.api|exists()":                       true,
				"server.api.$schema":                        "https://xmlui.org/schemas/v2/localsvr/api-schema.json",
				"server.api.version":                        2,
				"server.api.name":                           "User-definable XMLUI Local Server API",
				"server.api.base_path":                      "/api",
				"server.api.webroot":                        ".",
				"server.api.endpoints|exists()":             true,
				"server.api.endpoints|len()":                5,
				"server.api.endpoints.0.method":             "GET",
				"server.api.endpoints.0.path":               "/tasks/search/{project_id:int}",
				"server.api.endpoints.0.description":        "Search tasks within a given project (path param project_id + query-string param q)",
				"server.api.endpoints.0.query":              "SELECT t.id, t.title, t.status, t.priority, IFNULL(au.email,'') AS assignee_email FROM tasks t LEFT JOIN users au ON au.id = t.assignee_id WHERE t.project_id = :project_id AND (LOWER(t.title) LIKE LOWER('%' || :q || '%') OR LOWER(t.details) LIKE LOWER('%' || :q || '%')) ORDER BY t.priority DESC, t.id;",
				"server.api.endpoints.0.query_file":         "",
				"server.api.endpoints.0.params|len()":       3,
				"server.api.endpoints.0.cardinality":        "many",
				"server.api.endpoints.0.row_type":           "columns",
				"server.api.endpoints.0.column_types|len()": 5,
				"database|exists()":                         true,
				"database.$schema":                          "https://xmlui.org/schemas/v1/localsvr/db/sqlite3-schema.json",
				"database.version":                          1,
				"database.type":                             "sqlite3",
				"database.filepath":                         "./data.db",
				"database.on_open_sql|len()":                1,
				"database.busy_timeout":                     0,
				"database.journal_mode":                     "",
				"database.synchronous":                      "",
				"database.foreign_keys":                     "",
				"database.wal_autocheckpoint":               0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, _ := dt.Getwd()
			rootFix, css := SetupFixtures(t, SetupFixturesArgs{
				TestDataDir: dt.DirPathJoin(wd, testDataDir),
				UserFile:    "./user-config.localsvr.json",
				ProjectFile: "./project-config.localsvr.json",
			})
			defer rootFix.Cleanup()
			opts := cfgldr.NewOptions(cfgldr.OptionsArgs{})
			gotRc, err := cfgldr.LoadRootConfigV1(cfgldr.LoadRootConfigV1Args{
				Options:      opts,
				ConfigStores: css,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRootConfigV1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//println(string(gotRc.Bytes()))
			//return
			err = jsontest.TestJSON(gotRc.Bytes(), tt.want)
			require.NoError(t, err, "Config did not match expected values")
		})
	}
}

func writeJSONFile(file string, value any, filePerms, dirPerms os.FileMode) (err error) {
	var bytes []byte
	err = os.MkdirAll(filepath.Dir(file), dirPerms)
	if err != nil {
		err = fmt.Errorf("failed to create directory %s", filepath.Dir(file))
		goto end
	}

	bytes, err = jsonv2.Marshal(value, jsontext.WithIndent("\t"))
	if err != nil {
		err = fmt.Errorf("failed to marshal value of type '%T' to json", value)
		goto end
	}

	err = os.WriteFile(file, bytes, filePerms)
	if err != nil {
		err = fmt.Errorf("failed to write test file; %s", file)
		goto end
	}

end:
	return err
}
