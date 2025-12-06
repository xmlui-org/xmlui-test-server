package cfgldr

import (
	"context"
	jsonv2 "encoding/json/v2"
	"errors"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

type Context = context.Context

var _ Config = (*APIDescription)(nil)

// APIDescription is deprecated; use APIConfigV2
type APIDescription struct {
	APIVersion  string               `json:"apiVersion"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	BasePath    string               `json:"basePath"`
	Endpoints   []EndpointDefinition `json:"endpoints"`
	//pathRegexps map[string]*regexp.Regexp
}

func (d *APIDescription) Migrate() *APIConfigV2 {
	endpoints := make([]*APIEndpointV2, 0, len(d.Endpoints))
	for _, ep := range d.Endpoints {
		endpoints = append(endpoints, ep.Migrate()...)
	}
	return &APIConfigV2{
		Version:    APIConfigV2Version,
		Name:       d.Description,
		BasePath:   d.BasePath,
		Webroot:    localsvr.DefaultWebroot,
		Endpoints:  endpoints,
		SourceFile: "",
	}
}

func (*APIDescription) Config() {}

// EndpointDefinition models an APIConfig endpoint in v1 api.json schema
// Deprecated — use APIEndpoint instead
type EndpointDefinition struct {
	Path    string                      `json:"path"`
	Methods map[string]MethodDefinition `json:"methods"`
}

// Migrate migrates an EndpointDefinition to an APIEndpointV2
func (d *EndpointDefinition) Migrate() (eps []*APIEndpointV2) {
	eps = make([]*APIEndpointV2, 0, len(d.Methods))
	for name, m := range d.Methods {
		params := make(APIParamsV1, len(d.Methods))
		for i, p := range m.Params {
			params[i] = APIParamV1{
				NameSpec: p,
				Type:     string(sqlparams.AnyRowType),
			}
		}
		name = strings.ToUpper(name)
		eps = append(eps, NewAPIEndpointV2(name, d.Path, APIEndpointV2Args{
			Description: m.Description,
			Query:       m.SQL,
			QueryFile:   m.SQLFile,
			Params:      params,
			Cardinality: string(sqlparams.ManyRowsOrNone),
			RowType:     string(sqlparams.AnyRowType),
		}))
	}
	return eps
}

// MethodDefinition models an APIConfig endpoint method in v1 api.json schema
// Deprecated — use APIEndpoint instead
type MethodDefinition struct {
	Description string   `json:"description"`
	SQL         string   `json:"sql,omitempty"`
	SQLFile     string   `json:"sqlFile,omitempty"`
	Params      []string `json:"params,omitempty"`
}

func LoadAPIDescriptionFromFile(file dt.Filepath) (d *APIDescription, err error) {
	var data []byte
	data, err = file.ReadFile()
	if errors.Is(err, os.ErrNotExist) {
		goto end
	}
	if err != nil {
		err = NewErr(err, ErrReadFailed)
		goto end
	}
	d = &APIDescription{}
	err = jsonv2.Unmarshal(data, &d)
	if err != nil {
		d = nil
		err = NewErr(ErrParseFailed, err)
		goto end
	}

end:
	return d, err
}
