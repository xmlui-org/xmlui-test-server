package cfgldr

import (
	"errors"
	"fmt"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/go-sqlparams"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
)

const (
	SQLite3ConfigV1Version = 1
	SQLite3ConfigV1Schema  = "https://xmlui.org/schemas/v1/localsvr/db/sqlite3-schema.json"
)

func init() {
	registerDatabaseConfig(&SQLite3ConfigV1{})
}

var _ DatabaseConfig = (*SQLite3ConfigV1)(nil)

type SQLite3ConfigV1 struct {
	Schema         string                      `json:"$schema,omitempty"`
	Version        int                         `json:"version,omitempty"`
	Notes          []string                    `json:"@notes,omitempty"`
	Type           string                      `json:"type"`
	Filepath       string                      `json:"filepath"`
	Extensions     []*SQLite3ExtensionConfigV1 `json:"-"` // `json:"extensions"`
	OnOpenSQL      []string                    `json:"on_open_sql"`
	BusyTimeout    int                         `json:"busy_timeout"`
	JournalMode    string                      `json:"journal_mode"`
	Synchronous    string                      `json:"synchronous"`
	ForeignKeys    string                      `json:"foreign_keys"`
	AutoCheckpoint int                         `json:"wal_autocheckpoint"`
	AccessMode     int                         `json:"access_mode"`
	bootstrapSQL   []string
	sourceFile     string
}

func (c *SQLite3ConfigV1) Clone() DatabaseConfig {
	newCfg := *c

	newCfg.OnOpenSQL = cloneSlice(c.OnOpenSQL)
	newCfg.bootstrapSQL = cloneSlice(c.bootstrapSQL)
	newCfg.Extensions = make([]*SQLite3ExtensionConfigV1, len(c.Extensions))
	for i, ext := range c.Extensions {
		newCfg.Extensions[i] = ext.Clone()
	}

	return &newCfg
}

func (c *SQLite3ConfigV1) SetBootstrapQueries(queries []string) {
	c.bootstrapSQL = queries
}

func (c *SQLite3ConfigV1) BootstrapQueries() []string {
	return c.bootstrapSQL
}

func (c *SQLite3ConfigV1) OnOpenQueries() []string {
	return c.OnOpenSQL
}

func (c *SQLite3ConfigV1) SourceFile() string {
	return c.sourceFile
}

func NewSQLite3ConfigV1(filepath string) *SQLite3ConfigV1 {
	if len(filepath) != 0 && filepath[0] != '/' && filepath[0] != '.' {
		filepath = "./" + filepath
	}
	return &SQLite3ConfigV1{
		Schema:   SQLite3ConfigV1Schema,
		Version:  SQLite3ConfigV1Version,
		Notes:    make([]string, 0),
		Type:     string(SQLite3Database),
		Filepath: filepath,
	}
}
func (c *SQLite3ConfigV1) ConnectString() string {
	return c.Filepath
}

func (c *SQLite3ConfigV1) SetConnectString(cs string) {
	c.Filepath = cs
}

func (c *SQLite3ConfigV1) Port() int {
	return 0
}

func (c *SQLite3ConfigV1) AddDBExtension(ext DBExtensionConfig) {
	sExt, ok := ext.(*SQLite3ExtensionConfigV1)
	if !ok {
		panic(fmt.Sprintf("Cannot type assert a value of type %T to type %T for extension %s",
			ext, (*SQLite3ExtensionConfigV1)(nil),
			ext.ErrorName(),
		))
	}
	c.Extensions = append(c.Extensions, sExt)
}

func (c *SQLite3ConfigV1) DBExtensions() (exts []DBExtensionConfig) {
	exts = make([]DBExtensionConfig, len(c.Extensions))
	for i, ext := range c.Extensions {
		exts[i] = ext
	}
	return exts
}

func (c *SQLite3ConfigV1) DatabaseConfig() {}

func (c *SQLite3ConfigV1) Merge(base DatabaseConfig) DatabaseConfig {
	// Merge base into c, c takes precedence
	baseSQL3, ok := base.(*SQLite3ConfigV1)
	if !ok {
		return c // Can't merge different database types
	}

	// Filepath: Project wins if non-empty
	if c.Filepath == "" {
		c.Filepath = baseSQL3.Filepath
	}

	// OnOpenSQL: Project wins completely
	if len(c.OnOpenSQL) == 0 && len(baseSQL3.OnOpenSQL) > 0 {
		c.OnOpenSQL = baseSQL3.OnOpenSQL
	}

	// bootstrapSQL: Project wins if exists
	if len(c.bootstrapSQL) == 0 && len(baseSQL3.bootstrapSQL) > 0 {
		c.bootstrapSQL = baseSQL3.bootstrapSQL
	}

	// Extensions: Append - accumulate from both
	if len(baseSQL3.Extensions) > 0 {
		c.Extensions = append(c.Extensions, baseSQL3.Extensions...)
	}

	// Scalar fields: Project wins if non-zero/non-empty
	if c.BusyTimeout == 0 {
		c.BusyTimeout = baseSQL3.BusyTimeout
	}
	if c.JournalMode == "" {
		c.JournalMode = baseSQL3.JournalMode
	}
	if c.Synchronous == "" {
		c.Synchronous = baseSQL3.Synchronous
	}
	if c.ForeignKeys == "" {
		c.ForeignKeys = baseSQL3.ForeignKeys
	}
	if c.AutoCheckpoint == 0 {
		c.AutoCheckpoint = baseSQL3.AutoCheckpoint
	}
	if c.AccessMode == 0 {
		c.AccessMode = baseSQL3.AccessMode
	}

	// Notes: Skip (only for text files)

	return c
}

func (c *SQLite3ConfigV1) DatabaseType() DatabaseType {
	return SQLite3Database
}

func (c *SQLite3ConfigV1) AddExtension(ext *SQLite3ExtensionConfigV1, opts *Options) (err error) {
	err = ext.Normalize(cfgstore.NormalizeArgs{
		SourceFile: dt.Filepath(c.SourceFile()),
		Options:    opts,
		DirType:    0, // TODO Address this, if needed
	})
	if err != nil {
		goto end
	}
	c.Extensions = append(c.Extensions, ext)
end:
	return err
}

func (c *SQLite3ConfigV1) SetExtensions(exts []*SQLite3ExtensionConfigV1) {
	c.Extensions = exts
}

func (c *SQLite3ConfigV1) normalizeExtensions(args cfgstore.NormalizeArgs) (err error) {
	var errs []error
	if len(c.Extensions) == 0 {
		c.Extensions = make([]*SQLite3ExtensionConfigV1, 0)
		goto end
	}
	for _, ext := range c.Extensions {
		errs = AppendErr(errs, ext.Normalize(args))
	}
	err = CombineErrs(errs)
end:
	return err
}

func (c *SQLite3ConfigV1) normalizeConnectString(opts *Options) (err error) {
	cs := opts.appendToWebroot(opts.ConnectString, common.DefaultSQLite3Database)
	c.SetConnectString(cs)
	return err
}

func (c *SQLite3ConfigV1) Normalize(args cfgstore.NormalizeArgs) (err error) {
	var errs []error
	var opts *Options

	c.sourceFile = string(args.SourceFile)
	if c.Schema == "" {
		c.Schema = SQLite3ConfigV1Schema
	}
	if c.Version == 0 {
		c.Version = SQLite3ConfigV1Version
	}
	if c.Type == "" {
		c.Type = string(SQLite3Database)
	}
	if len(c.bootstrapSQL) == 0 {
		c.bootstrapSQL = make([]string, 0)
	}
	if len(c.OnOpenSQL) == 0 {
		c.OnOpenSQL = make([]string, 0)
	}
	opts, err = dtx.AssertType[*Options](args.Options)
	if err != nil {
		goto end
	}
	if opts.DBAccessMode != int(sqlparams.UnspecifiedDBAccessMode) {
		c.AccessMode = opts.DBAccessMode
	}
	if c.AccessMode == int(sqlparams.UnspecifiedDBAccessMode) {
		c.AccessMode = DefaultDBAccessMode
	}
	errs = AppendErr(errs, c.normalizeConnectString(opts))
	errs = AppendErr(errs, c.normalizeExtensions(args))
	err = CombineErrs(errs)
end:
	if err != nil {
		err = WithErr(err,
			ErrFailedToNormalize,
			"source_config", args.SourceFile,
		)
	}
	return err
}

var _ DBExtensionConfig = (*SQLite3ExtensionConfigV1)(nil)

type SQLite3ExtensionConfigV1 struct {
	Notes        []string          `json:"@notes,omitempty"`
	Id           string            `json:"id"`
	Version      string            `json:"version"`
	Name         string            `json:"name"`
	DocsURL      string            `json:"docs_url"`
	RepoURL      string            `json:"repo_url"`
	DownloadURLs []string          `json:"download_urls"`
	Filepath     string            `json:"filepath"` // Absolute or relative filepath, defaults to well-known directory structure
	LoadOrder    int               `json:"load_order"`
	EntryPoint   string            `json:"entry_point"`
	DependsOn    []string          `json:"depends_on"`
	SHA256s      map[string]string `json:"sha256s"`
	OnFailure    string            `json:"on_failure"` // 'error','warn','ignore'
	OnLoadSQL    []string          `json:"post_load_sql"`
	EnvVars      map[string]string `json:"env_vars"`
	VarScope     string            `json:"vars_scope"` // 'load' or 'app'
	AllowVTable  bool              `json:"allow_vtable"`
	SourceFile   string            `json:"-"`
}

func (c *SQLite3ExtensionConfigV1) ErrorName() string {
	return c.Name
}

func (*SQLite3ExtensionConfigV1) DBExtensionConfig() {}

func (c *SQLite3ExtensionConfigV1) AddDownloadURL(url string) {
	c.DownloadURLs = append(c.DownloadURLs, url)
}
func (c *SQLite3ExtensionConfigV1) AddOnLoadSQL(sql string) {
	c.OnLoadSQL = append(c.OnLoadSQL, sql)
}
func (c *SQLite3ExtensionConfigV1) AddDependsOn(do string) {
	c.DependsOn = append(c.DependsOn, do)
}
func (c *SQLite3ExtensionConfigV1) AddSHA256(name, value string) {
	c.SHA256s[name] = value
}
func (c *SQLite3ExtensionConfigV1) AddEnvVar(name, value string) {
	c.EnvVars[name] = value
}

var (
	ErrFailedToNormalize         = errors.New("failed to normalize")
	ErrInsufficientConfiguration = errors.New("insufficient configuration")
	ErrMustSpecifyOneOf          = errors.New("must specify one of")
)

func (c *SQLite3ExtensionConfigV1) Normalize(args cfgstore.NormalizeArgs) (err error) {
	var filePath dt.Filepath
	c.SourceFile = string(args.SourceFile)

	switch {
	case c.Filepath != "":
		filePath = dt.Filepath(c.Filepath)
	case len(c.DownloadURLs) != 0:
		c.Filepath = c.DownloadURLs[0] // TODO: Dervive filename from download URL
		panic("IMPLEMENT ME")
	default:
		err = NewErr(
			ErrInsufficientConfiguration,
			ErrMustSpecifyOneOf,
			"choices", []string{"filepath", "a download_url"},
		)
		goto end
	}
	if c.Id == "" {
		base := filePath.Base()
		ext := base.Ext()
		c.Id = string(base)[:len(base)-len(ext)]
	}
	if c.Notes == nil {
		c.Notes = make([]string, 0)
	}
	if c.Name == "" {
		c.Name = c.Id
	}
	if c.Version == "" {
		c.Version = common.UnknownVersion
	}
	if c.EntryPoint == "" {
		c.EntryPoint = common.DefaultSQLite3ExtensionEntryPoint
	}
	if c.OnFailure == "" {
		c.OnFailure = common.DefaultOnFailurePolicy
	}
	if c.VarScope == "" {
		c.VarScope = common.DefaultVarScope
	}
	if c.DownloadURLs == nil {
		c.DownloadURLs = make([]string, 0)
	}
	if c.DependsOn == nil {
		c.DependsOn = make([]string, 0)
	}
	if c.OnLoadSQL == nil {
		c.OnLoadSQL = make([]string, 0)
	}
	if c.SHA256s == nil {
		c.SHA256s = make(map[string]string)
	}
	if c.EnvVars == nil {
		c.EnvVars = make(map[string]string)
	}
end:
	if err != nil {
		err = WithErr(err,
			ErrFailedToNormalize,
			"sqlite3_extension", c.Name,
		)
	}
	return err
}

func (c *SQLite3ExtensionConfigV1) Clone() (ext *SQLite3ExtensionConfigV1) {
	ext = &SQLite3ExtensionConfigV1{}
	*ext = *c
	ext.DownloadURLs = cloneSlice(c.DownloadURLs)
	ext.DependsOn = cloneSlice(c.DependsOn)
	ext.OnLoadSQL = cloneSlice(c.OnLoadSQL)
	ext.EnvVars = cloneMap(c.EnvVars)
	ext.SHA256s = cloneMap(c.SHA256s)
	return ext
}
