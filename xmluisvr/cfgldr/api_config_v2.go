package cfgldr

import (
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

const (
	APIConfigV2Version = 2
	APIConfigV2Schema  = "https://xmlui.org/schemas/v2/localsvr/api-schema.json"
)

var _ APIConfig = (*APIConfigV2)(nil)

type APIConfigV2 struct {
	Schema     string           `json:"$schema"`
	Version    int              `json:"version"`
	Notes      []string         `json:"@notes,omitempty"`
	Name       string           `json:"name"`
	BasePath   string           `json:"base_path"`
	Webroot    string           `json:"webroot"`
	Endpoints  []*APIEndpointV2 `json:"endpoints"`
	SourceFile string           `json:"-"`
}

func (c *APIConfigV2) IsNil() (isNil bool) {
	var v reflect.Value

	isNil = true
	if c == nil {
		goto end
	}
	v = reflect.ValueOf(c)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		isNil = v.IsNil()
	default:
		isNil = false
	}
end:
	return isNil
}

func NewAPIConfigV2(webroot string) *APIConfigV2 {
	return &APIConfigV2{
		Schema:     APIConfigV2Schema,
		Version:    APIConfigV2Version,
		Notes:      make([]string, 0),
		Name:       fmt.Sprintf("User-definable %s APIConfig", common.AppName),
		BasePath:   common.DefaultAPIBasePath,
		Webroot:    webroot,
		Endpoints:  make([]*APIEndpointV2, 0),
		SourceFile: common.DefaultAPIConfigFile,
	}
}

func (*APIConfigV2) Config() {}

func (c *APIConfigV2) Merge(base *APIConfigV2) *APIConfigV2 {
	// Merge base into c, c takes precedence
	if c.Name == "" {
		c.Name = base.Name
	}
	if c.BasePath == "" {
		c.BasePath = base.BasePath
	}
	if c.Webroot == "" {
		c.Webroot = base.Webroot
	}
	// Endpoints: If Project has ANY endpoints, use ONLY Project's endpoints
	// Otherwise use CLI's endpoints
	if len(c.Endpoints) == 0 && len(base.Endpoints) > 0 {
		c.Endpoints = base.Endpoints
	}
	// Notes: Skip merging (only for text files, per user)
	return c
}

func (c *APIConfigV2) normalizeEndpoints(args cfgstore.NormalizeArgs) (err error) {
	var errs []error
	if c.Endpoints == nil {
		c.Endpoints = make([]*APIEndpointV2, 0)
	}
	if len(c.Endpoints) == 0 {
		goto end
	}
	for _, ep := range c.Endpoints {
		errs = AppendErr(errs, ep.Normalize(args))
	}
	err = CombineErrs(errs)
end:
	return err
}

func (c *APIConfigV2) normalizeWebroot(args cfgstore.NormalizeArgs) (err error) {
	var opts *Options
	opts, err = dtx.AssertType[*Options](args.Options)
	if err != nil {
		goto end
	}
	switch {
	case filepath.Clean(c.Webroot) == ".":
		c.Webroot = opts.Webroot
	case !filepath.IsAbs(c.Webroot):
		c.Webroot = filepath.Join(opts.Webroot, c.Webroot)
	}
end:
	return err
}

func (c *APIConfigV2) Normalize(args cfgstore.NormalizeArgs) (err error) {
	var errs []error
	c.SourceFile = string(args.SourceFile)
	if c.Schema == "" {
		c.Schema = APIConfigV2Schema
	}
	if c.Version == 0 {
		c.Version = APIConfigV2Version
	}
	if c.BasePath == "" {
		c.BasePath = common.DefaultAPIBasePath
	}
	errs = AppendErr(errs, c.normalizeWebroot(args))
	errs = AppendErr(errs, c.normalizeEndpoints(args))

	err = CombineErrs(errs)
	return err
}

func (c *APIConfigV2) AddEndpoint(endpoint *APIEndpointV2) {
	c.Endpoints = append(c.Endpoints, endpoint)
}

func (c *APIConfigV2) Migrate(oldCfg Config) (newCfg *APIConfigV2) {
	v1 := oldCfg.(*APIDescription)
	common.Noop(v1) // TODO Implement migration
	return new(APIConfigV2)
}

func LoadAPIConfigV2(apiFile dt.Filepath) (c *APIConfigV2, err error) {
	var data []byte
	if apiFile == "" {
		goto end
	}
	data, err = apiFile.ReadFile()
	if errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("invalid APIConfig description file: %w", err)
		goto end
	}
	if err != nil {
		err = WithErr(err, ErrReadFailed)
		goto end
	}
	c = new(APIConfigV2)
	err = jsonv2.Unmarshal(data, c)
	if err != nil {
		c = nil
		// TODO: Provide user better feedback as to what actually failed.
		err = NewErr(ErrParseFailed, err)
		goto end
	}
end:
	return c, err
}
