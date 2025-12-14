package cfgldr

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// APIEndpointV2 is the main endpoint struct using JSONV2 inline to flatten the JSON
type APIEndpointV2 struct {
	APIEndpointBase `json:",inline"`
	Params          APIParamsMapper `json:"params"`
	paramsType      reflect.Type
}

// APIEndpointBase contains all the non-polymorphic properties for APIEndpointV2
type APIEndpointBase struct {
	Notes         []string `json:"@notes,omitempty"`
	Method        string   `json:"method"`
	Path          string   `json:"path"`
	Description   string   `json:"description"`
	Query         string   `json:"query"`
	QueryFile     string   `json:"query_file"`
	configDir     string
	queryFilepath string
	Cardinality   string   `json:"cardinality"`  // 'one' or 'many'
	RowType       string   `json:"row_type"`     // 'int', 'real','string','json','columns'
	ColumnTypes   []string `json:"column_types"` // used when row_type="columns"
}

func (ep APIEndpointBase) Endpoint() string {
	var method string
	var path string
	if ep.Method == "" {
		method = string(localsvr.ANYMethod)
	} else {
		method = ep.Method
	}
	switch {
	case ep.Path == "":
		path = "/"
	case ep.Path[0] != '/':
		path = fmt.Sprintf("./%s", ep.Path)
	default:
		path = ep.Path
	}
	return fmt.Sprintf("%s %s", method, path)
}

type APIEndpointV2Args struct {
	Notes       []string
	Description string
	Query       string
	QueryFile   string
	Params      APIParamsMapper
	Cardinality string
	RowType     string
	ColumnTypes []string
	ParamsType  reflect.Type
}

func NewAPIEndpointV2(method, path string, args APIEndpointV2Args) *APIEndpointV2 {
	if args.Notes == nil {
		args.Notes = make([]string, 0)
	}
	if args.Params == nil {
		args.Params = APIParamsV1{}
	}
	if args.ColumnTypes == nil {
		args.ColumnTypes = make([]string, 0)
	}
	if args.ParamsType == nil {
		args.ParamsType = reflect.TypeOf(([]APIParamV1)(nil))
	}
	switch args.ParamsType {
	case reflect.TypeOf(([]APIParamV1)(nil)):
	case reflect.TypeOf((*APIParamsMap)(nil)):
	default:
		panic(fmt.Sprintf("Unsupported Parameters Type '%T' for endpoint %s %s'", args.ParamsType, method, path))
	}
	return &APIEndpointV2{
		APIEndpointBase: APIEndpointBase{
			Notes:       args.Notes,
			Method:      method,
			Path:        path,
			Description: args.Description,
			Query:       args.Query,
			QueryFile:   args.QueryFile,
			Cardinality: args.Cardinality,
			RowType:     args.RowType,
			ColumnTypes: args.ColumnTypes,
		},
		Params:     args.Params,
		paramsType: args.ParamsType,
	}
}

func (ep *APIEndpointV2) normalizeQueryFile(args cfgstore.NormalizeArgs) (err error) {
	var exists bool
	if ep.QueryFile == "" {
		goto end
	}
	switch args.DirType {
	case cfgstore.CLIConfigDirType, cfgstore.AppConfigDirType:
		ep.queryFilepath = filepath.Join(ep.configDir, ep.QueryFile)
	case cfgstore.ProjectConfigDirType:
		var opts *Options
		opts, err = dtx.AssertType[*Options](args.Options)
		if err != nil {
			goto end
		}
		ep.queryFilepath = filepath.Join(opts.Webroot, ep.QueryFile)
	case cfgstore.UnspecifiedConfigDirType:
		// Just here to stop GoLand from complaining about missing case statements
	}
	exists, _ = dt.Filepath(ep.queryFilepath).Exists()
	// If not, remove it as this is an optional file (I think)
	if !exists {
		// TODO: Consider keeping track of QueryFiles that are specified but do not exist
		//  in any of the config stores (project or CLI/App config store). We cannot throw
		//  an error if missing unless we start tracking all config stores because if it
		//  is found it should only be found in one config store.
		ep.queryFilepath = ""
	}
end:
	return err
}

func (ep *APIEndpointV2) Normalize(args cfgstore.NormalizeArgs) error {
	if ep.Description == "" {
		ep.Description = ep.Endpoint()
	}
	if ep.Cardinality == "" {
		ep.Cardinality = string(sqlparams.DefaultCardinality)
	}
	if ep.RowType == "" {
		ep.RowType = string(sqlparams.DefaultRowType)
	}
	if ep.Params == nil {
		ep.Params = APIParamsV1{}
	}
	if ep.paramsType == nil {
		ep.paramsType = reflect.TypeOf(([]APIParamV1)(nil))
	}
	if ep.configDir == "" {
		ep.configDir = string(args.SourceFile.Dir())
	}
	return ep.normalizeQueryFile(args)
}

func (ep *APIEndpointV2) IsMapFormat() bool {
	return ep.paramsType == reflect.TypeOf((*APIParamsMap)(nil))
}

func (ep *APIEndpointV2) MarshalJSON() (json []byte, err error) {
	var apiParams APIParamsV1
	var ok bool

	// Create temporary struct for marshaling
	var temp struct {
		APIEndpointBase `json:",inline"`
		Params          any `json:"params"`
	}
	// Copy base fields
	temp.APIEndpointBase = ep.APIEndpointBase

	marshalFunc := func() ([]byte, error) {
		return jsonv2.Marshal(temp, jsontext.WithIndent("  "))
	}

	if !ep.IsMapFormat() {
		// Keep as array format
		temp.Params = ep.Params
		json, err = marshalFunc()
		goto end
	}

	// Convert params to original format based on paramsType
	apiParams, ok = ep.Params.(APIParamsV1)
	if !ok {
		err = NewErr(ErrAPIParamsIsAnInvalidDataType,
			"endpoint", ep.Endpoint(),
			fmt.Errorf("data_type=%T", ep.Params),
		)
		goto end
	}

	temp.Params = apiParams.APIParamsMap()
	json, err = marshalFunc()

end:
	return json, err
}

func (ep *APIEndpointV2) UnmarshalJSON(data []byte) (err error) {
	var isMap bool
	var errs []error

	// Create a temporary struct that matches RootConfigV1 but with DBConfig as RawMessage
	var temp struct {
		APIEndpointBase `json:",inline"`
		Params          jsontext.Value `json:"params"`
	}

	var params []APIParamV1
	var paramsMap APIParamsMap

	err = jsonv2.Unmarshal(data, &temp)
	if err != nil {
		goto end
	}

	ep.APIEndpointBase = temp.APIEndpointBase

	if []byte(temp.Params) == nil {
		ep.Params = APIParamsV1{}
		ep.paramsType = reflect.TypeOf(([]APIParamV1)(nil))
		goto end
	}

	err = jsonv2.Unmarshal(temp.Params, &params)
	if err != nil {
		isMap = true
		err = jsonv2.Unmarshal(temp.Params, &paramsMap)
	}
	if err != nil {
		goto end
	}
	if !isMap {
		ep.Params = APIParamsV1(params)
		ep.paramsType = reflect.TypeOf(([]APIParamV1)(nil))
		goto end
	}
	for name, spec := range paramsMap.Iterator() {
		if name != "" && name[0] == '@' {
			// Ignore comments
			continue
		}
		param, err := ParseAPIParamV1(string(name), string(spec))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		params = append(params, param)
	}
	ep.Params = APIParamsV1(params)
	ep.paramsType = reflect.TypeOf((*APIParamsMap)(nil))
	err = CombineErrs(errs)

end:
	return err
}

func (ep *APIEndpointV2) GetQuery() (q string, err error) {
	var queryBytes []byte

	q = ep.Query

	if q != "" && ep.QueryFile != "" {
		err = NewErr(ErrEitherQueryOrQueryFile,
			"endpoint", ep.Endpoint(),
			"query", leftN(ep.QueryFile, 50),
			"query_file", ep.QueryFile,
		)
	}
	// Check if SQL should be loaded from a file
	if ep.QueryFile == "" {
		// Use the inline SQL from the APIConfig definition
		goto end
	}
	ep.queryFilepath = ep.GetQueryFilepath()

	// Determine the APIConfig description file's directory to make relative paths work

	// Read the SQL file
	queryBytes, err = os.ReadFile(ep.queryFilepath)
	if err != nil {
		err = NewErr(ErrFailedToReadQueryFile,
			"endpoint", ep.Endpoint(),
			"query", leftN(ep.QueryFile, 50),
			"query_file", ep.QueryFile,
			err,
		)
		goto end
	}

	q = string(queryBytes)
end:
	return q, err
}

func (ep *APIEndpointV2) GetQueryFilepath() string {
	if ep.configDir == "" {
		panic(fmt.Sprintf("Config Directory not set for %s", ep.Endpoint()))
	}
	if ep.queryFilepath != "" {
		goto end
	}
	// Build the SQL file path relative to the APIConfig description file
	ep.queryFilepath = filepath.Join(ep.configDir, ep.QueryFile)
end:
	return ep.queryFilepath
}

func leftN(s string, n int) string {
	if len(s) < n {
		goto end
	}
	s = s[:n]
end:
	return s
}
