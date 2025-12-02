package sqlite3pkg

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mattn/go-sqlite3"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/common"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
)

type (
	EntryPoint string
	SQLQuery   string
)

var _ dbpkg.DBExtension = (*Extension)(nil)

type Extension struct {
	id      common.ExtensionId
	version common.Version
	name    string
	//docsURL      common.FullURL
	//repoURL      common.FullURL
	downloadURLs []common.FullURL
	filePath     dt.Filepath // Absolute or relative filepath, defaults to well-known directory structure
	loadOrder    common.LoadOrder
	entryPoint   EntryPoint
	dependsOn    []DependsOn
	sha256s      map[common.OSArch]common.SHA256
	onFailure    common.OnFailure
	preLoadSQL   []SQLQuery
	postLoadSQL  []SQLQuery
	envVars      common.EnvironmentVars
	allowVTable  bool
	varScope     EnvVarScope // load or app
}
type ExtensionArgs struct {
}

func NewExtension(filePath dt.Filepath, args ExtensionArgs) *Extension {
	name := filepath.Base(string(filePath))
	name = name[:len(name)-len(filepath.Ext(name))]
	return &Extension{
		id:           common.ExtensionId(name),
		version:      common.UnknownVersion,
		name:         name,
		downloadURLs: make([]common.FullURL, 0),
		filePath:     filePath,
		loadOrder:    0,
		entryPoint:   common.DefaultSQLite3ExtensionEntryPoint,
		dependsOn:    make([]DependsOn, 0),
		sha256s:      make(map[common.OSArch]common.SHA256),
		onFailure:    common.DefaultOnFailurePolicy,
		preLoadSQL:   make([]SQLQuery, 0),
		postLoadSQL:  make([]SQLQuery, 0),
		envVars:      make(common.EnvironmentVars),
		allowVTable:  false,
		varScope:     common.DefaultVarScope,
	}
}

func (ext *Extension) Load(conn *sqlite3.SQLiteConn, db *SQLite3) (err error) {
	var path string

	// Important: do NOT globally enable load_extension via SQL unless explicitly requested.
	path, err = ext.Resolve(db)
	if err != nil {
		goto end
	}
	if path == "" {
		goto end
	}
	err = ext.runOnEventSQL(conn, "preLoad", ext.preLoadSQL)
	if err != nil {
		goto end
	}
	err = conn.LoadExtension(path, "")
	if err != nil {
		return fmt.Errorf("LoadExtension(%s): %w", path, err)
	}
	err = ext.runOnEventSQL(conn, "postLoad", ext.postLoadSQL)
	if err != nil {
		goto end
	}
end:
	return err
}

func (ext *Extension) runOnEventSQL(conn *sqlite3.SQLiteConn, et string, onEventSQL []SQLQuery) (err error) {
	var errs []error
	for _, sqlQuery := range onEventSQL {
		q := strings.TrimSpace(string(sqlQuery))
		if q == "" {
			continue
		}
		_, err = conn.Exec(q, nil)
		if err != nil {
			errs = append(errs, WithErr(ErrInEventQueryForSQLite3Extension,
				"event_query", et,
				"extension", ext.Name(),
				err,
			))
		}
	}
	return CombineErrs(errs)
}

func (ext *Extension) Name() string {
	return ext.name
}

func (ext *Extension) DBExtension() {}

// Resolve decides which file to load and where it came from.
func (ext *Extension) Resolve(db *SQLite3) (path string, err error) {
	panic("IMPLEMENT EXTENSION RESOLVER")
}

//func configRoot() (root string, err error) {
//	root, err = os.UserConfigDir()
//	if err != nil {
//		goto end
//	}
//	root = filepath.Join(root, "xmlui", "sqlite", "exts")
//end:
//	return root, err
//}
//
//func platformKey() string { return runtime.GOOS + "-" + runtime.GOARCH }
//
//func sha256Hex(b []byte) string {
//	sum := sha256.Sum256(b)
//	return hex.EncodeToString(sum[:])
//}
//
//// findUserVersions returns available versions in config dir for an extension.
//func findUserVersions(name string) ([]string, error) {
//	root, err := configRoot()
//	if err != nil {
//		return nil, err
//	}
//	base := filepath.Join(root, name)
//	d, err := os.ReadDir(base)
//	if err != nil {
//		if errors.Is(err, os.ErrNotExist) {
//			return nil, nil
//		}
//		return nil, err
//	}
//	var vs []string
//	for _, e := range d {
//		if e.IsDir() {
//			vs = append(vs, e.Name())
//		}
//	}
//	sort.Strings(vs) // lexical; OK if you use simple semver
//	return vs, nil
//}
//
//func verifySHA256(path, want string) error {
//	if want == "" {
//		return nil
//	}
//	b, err := os.ReadFile(path)
//	if err != nil {
//		return err
//	}
//	got := sha256Hex(b)
//	if !strings.EqualFold(got, want) {
//		return fmt.Errorf("sha256 mismatch: got %s want %s", got, want)
//	}
//	return nil
//}
//
//func extDir(name, version string) (string, error) {
//	root, err := configRoot()
//	if err != nil {
//		return "", err
//	}
//	return filepath.Join(root, name, version, platformKey()), nil
//}
