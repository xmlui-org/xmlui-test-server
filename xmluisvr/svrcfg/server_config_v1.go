package cfgldr

import (
	"github.com/mikeschinkel/go-cfgstore"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

const (
	ServerConfigV1Version = 1
	ServerConfigV1Schema  = "https://xmlui.org/schemas/v1/localsvr/server-schema.json"
)

type ServerConfig interface {
	ServerConfig()
}

var _ ServerConfig = (*ServerConfigV1)(nil)

// ServerConfigV1 represents server-level configuration
type ServerConfigV1 struct {
	Schema     string       `json:"$schema,omitempty"`
	Version    int          `json:"version,omitempty"`
	Notes      []string     `json:"@notes,omitempty"`
	Host       string       `json:"host,omitempty"`
	Port       int          `json:"port,omitempty"`
	APIConfig  *APIConfigV2 `json:"api,omitempty"`
	SourceFile string       `json:"-"`
}

func (*ServerConfigV1) ServerConfig() {}

func (c *ServerConfigV1) Merge(base *ServerConfigV1) *ServerConfigV1 {
	// Merge base into c, c takes precedence
	if c.Host == "" {
		c.Host = base.Host
	}
	if c.Port == 0 {
		c.Port = base.Port
	}
	if c.APIConfig == nil {
		c.APIConfig = base.APIConfig
	} else if base.APIConfig != nil {
		c.APIConfig = c.APIConfig.Merge(base.APIConfig)
	}
	// Notes: append base notes to c's notes (accumulate)
	if len(base.Notes) > 0 {
		c.Notes = append(c.Notes, base.Notes...)
	}
	return c
}

func (c *ServerConfigV1) Normalize(args cfgstore.NormalizeArgs) (err error) {
	c.Schema = ServerConfigV1Schema
	c.Version = ServerConfigV1Version
	c.SourceFile = string(args.SourceFile)
	if c.Host == "" {
		c.Host = localsvr.DefaultServerHost
	}
	if c.Port == 0 {
		c.Port = localsvr.DefaultServerPort
	}
	err = c.APIConfig.Normalize(args)
	return err
}

type ServerConfigV1Args struct {
	Port  int
	API   *APIConfigV2
	Notes []string
}

func NewServerConfigV1(host string, args ServerConfigV1Args) *ServerConfigV1 {
	return &ServerConfigV1{
		Schema:    ServerConfigV1Schema,
		Version:   ServerConfigV1Version,
		Host:      host,
		Port:      args.Port,
		APIConfig: args.API,
		Notes:     args.Notes,
	}
}
