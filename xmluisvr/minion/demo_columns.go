package minion

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mikeschinkel/go-dt"
)

// DemoColumn identifies a specific demo column
type DemoColumn string

const (
	DemoColumnIndex     DemoColumn = "index"
	DemoColumnName      DemoColumn = "name"
	DemoColumnDomain    DemoColumn = "domain"
	DemoColumnPath      DemoColumn = "path"
	DemoColumnFullname  DemoColumn = "fullname"
	DemoColumnOrg       DemoColumn = "org"
	DemoColumnRepo      DemoColumn = "repo"
	DemoColumnType      DemoColumn = "type"
	DemoColumnRef       DemoColumn = "ref"
	DemoColumnTypedRef  DemoColumn = "typed_ref"
	DemoColumnSourceURL DemoColumn = "source_url"
	DemoColumnInstall   DemoColumn = "install_path"
	DemoColumnInstalled DemoColumn = "installed"
	DemoColumnAge       DemoColumn = "age"
	DemoColumnSize      DemoColumn = "size"
	DemoColumnDesc      DemoColumn = "description"
	DemoColumnValid     DemoColumn = "valid"
)

// ColumnInfo provides info for column value functions
type ColumnInfo struct {
	TimeNow    time.Time
	TimeFormat dt.TimeFormat
	HomeDir    dt.DirPath
	DemosDir   dt.DirPath
}

// ColumnMeta holds metadata about a column
type ColumnMeta struct {
	ID          DemoColumn
	Header      string
	JSONField   string
	Description string
	Alignment   text.Align
	ValueFunc   func(d *Demo, ctx *ColumnInfo) string
}

// This is a total hack
var columnIndex int

// columnRegistry is the package-scoped registry of all columns
var columnRegistry = map[DemoColumn]*ColumnMeta{
	DemoColumnName: {
		ID:          DemoColumnName,
		Header:      "DEMO",
		JSONField:   "demo",
		Description: "Demo descritor (e.g., GitHub repo, or URL path)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) (s string) {
			switch d.RefType {
			case BranchRefType, TagRefType, HashRefType:
				s = fmt.Sprintf("%s/%s", d.Org, d.Repo)
			case URLRefType:
				fallthrough
			default:
				s = string(d.SourceURL)
			}
			return s
		},
	},
	DemoColumnIndex: {
		ID:          DemoColumnIndex,
		Header:      "#",
		JSONField:   "index",
		Description: "Ordering index in the table",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			columnIndex++
			return strconv.Itoa(columnIndex)
		},
	},
	DemoColumnDomain: {
		ID:          DemoColumnDomain,
		Header:      "DOMAIN",
		JSONField:   "domain",
		Description: "Source domain (e.g., github.com, example.com)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.Domain)
		},
	},
	DemoColumnPath: {
		ID:          DemoColumnPath,
		Header:      "PATH",
		JSONField:   "path",
		Description: "Host-relative path to the demo",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.Path)
		},
	},
	DemoColumnFullname: {
		ID:          DemoColumnFullname,
		Header:      "FULLNAME",
		JSONField:   "fullname",
		Description: "Full name of the demo (org/repo or domain/path)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.FullName())
		},
	},
	DemoColumnOrg: {
		ID:          DemoColumnOrg,
		Header:      "ORG",
		JSONField:   "org",
		Description: "Organization (GitHub) or host (URL)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.Org)
		},
	},
	DemoColumnRepo: {
		ID:          DemoColumnRepo,
		Header:      "REPO",
		JSONField:   "repo",
		Description: "Repository name (GitHub) or last path segment (URL)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.Repo)
		},
	},
	DemoColumnType: {
		ID:          DemoColumnType,
		Header:      "TYPE",
		JSONField:   "type",
		Description: "Ref type (branch, tag, hash, or url)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.RefType)
		},
	},
	DemoColumnRef: {
		ID:          DemoColumnRef,
		Header:      "REF",
		JSONField:   "ref",
		Description: "Ref value (branch/tag/hash name or empty for URL)",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.Ref)
		},
	},
	DemoColumnTypedRef: {
		ID:          DemoColumnTypedRef,
		Header:      "REF",
		JSONField:   "typed_ref",
		Description: "Typed ref (e.g., 'branch: demo', 'tag: v1.2.3')",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			var typed string

			if d.Ref != "" {
				typed = fmt.Sprintf("%s: %s", d.RefType, d.Ref)
			} else {
				typed = string(d.RefType)
			}

			return typed
		},
	},
	DemoColumnSourceURL: {
		ID:          DemoColumnSourceURL,
		Header:      "SOURCE URL",
		JSONField:   "source_url",
		Description: "ZIP URL used at install time",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return string(d.SourceURL)
		},
	},
	DemoColumnInstall: {
		ID:          DemoColumnInstall,
		Header:      "INSTALL PATH",
		JSONField:   "install_path",
		Description: "Full filesystem path where demo is installed",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			var path dt.DirPath

			path = d.InstallPath
			if ctx.HomeDir != "" && d.InstallPath.HasPrefix(ctx.HomeDir) {
				path = "~" + d.InstallPath[len(ctx.HomeDir):]
			}

			return string(path)
		},
	},
	DemoColumnInstalled: {
		ID:          DemoColumnInstalled,
		Header:      "INSTALLED",
		JSONField:   "installed",
		Description: "Date/time when demo was installed or updated",
		Alignment:   text.AlignRight,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			var dateStr string

			if !d.Installed.IsZero() {
				dateStr = d.Installed.Format(string(ctx.TimeFormat))
			}

			return dateStr
		},
	},
	DemoColumnAge: {
		ID:          DemoColumnAge,
		Header:      "AGE",
		JSONField:   "age",
		Description: "Human-readable time since installation (e.g., 2d, 3h)",
		Alignment:   text.AlignRight,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return formatAge(d.Installed, ctx.TimeNow)
		},
	},
	DemoColumnSize: {
		ID:          DemoColumnSize,
		Header:      "SIZE",
		JSONField:   "size",
		Description: "Directory size in human-readable format",
		Alignment:   text.AlignRight,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return formatSize(d.Size)
		},
	},
	DemoColumnDesc: {
		ID:          DemoColumnDesc,
		Header:      "DESCRIPTION",
		JSONField:   "description",
		Description: "Description from README or directory name",
		Alignment:   text.AlignLeft,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			return d.Description
		},
	},
	DemoColumnValid: {
		ID:          DemoColumnValid,
		Header:      "VALID",
		JSONField:   "valid",
		Description: "Validation status (✓ valid / ✗ invalid)",
		Alignment:   text.AlignCenter,
		ValueFunc: func(d *Demo, ctx *ColumnInfo) string {
			if d.Valid() {
				return "✓"
			}
			return "✗"
		},
	},
}

// columnAliases maps user-friendly names to column IDs
var columnAliases = map[string]DemoColumn{
	"branch": DemoColumnRef,
	"url":    DemoColumnSourceURL,
	"dir":    DemoColumnInstall,
}

// DefaultColumns is the default column selection
var DefaultColumns = []DemoColumn{
	DemoColumnIndex,
	DemoColumnDomain,
	DemoColumnName,
	DemoColumnTypedRef,
	DemoColumnInstalled,
	DemoColumnValid,
}

// ResolveColumn resolves a column name to a DemoColumn ID, handling aliases
func ResolveColumn(name string) (col DemoColumn, err error) {
	var ok bool

	name = strings.ToLower(strings.TrimSpace(name))

	// Check if it's a known column directly
	col = DemoColumn(name)
	if _, ok = columnRegistry[col]; ok {
		goto end
	}

	// Check if it's an alias
	if col, ok = columnAliases[name]; ok {
		goto end
	}

	// Not found
	err = fmt.Errorf("unknown column: %s", name)

end:
	return col, err
}

// GetAllColumns returns all valid column IDs in sorted order
func GetAllColumns() (cols []DemoColumn) {
	var col DemoColumn

	cols = make([]DemoColumn, 0, len(columnRegistry))

	for col = range columnRegistry {
		cols = append(cols, col)
	}

	sort.Slice(cols, func(i, j int) bool {
		return string(cols[i]) < string(cols[j])
	})

	return cols
}

// GetColumnMeta returns metadata for a column
func GetColumnMeta(col DemoColumn) (meta *ColumnMeta) {
	meta = columnRegistry[col]
	return meta
}

// ValidateColumns validates a slice of column names and returns the resolved columns
func ValidateColumns(names []string) (cols []DemoColumn, err error) {
	var i int
	var name string
	var col DemoColumn

	cols = make([]DemoColumn, 0, len(names))

	for i, name = range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		col, err = ResolveColumn(name)
		if err != nil {
			err = fmt.Errorf("invalid column at position %d: %w", i, err)
			goto end
		}

		cols = append(cols, col)
	}

end:
	return cols, err
}

// FormatInvalidColumnError formats a user-friendly error message for invalid columns
func FormatInvalidColumnError(errMsg string) (formatted string) {
	var cols []DemoColumn
	var col DemoColumn
	var buf strings.Builder

	buf.WriteString(errMsg)
	buf.WriteString("\n\nAvailable columns:\n")

	cols = GetAllColumns()

	for _, col = range cols {
		buf.WriteString("  - ")
		buf.WriteString(string(col))
		buf.WriteString("\n")
	}

	formatted = buf.String()

	return formatted
}

// formatAge returns a human-readable age string from two times
func formatAge(installed time.Time, now time.Time) (age string) {
	var duration time.Duration

	if installed.IsZero() {
		return ""
	}

	duration = now.Sub(installed)

	switch {
	case duration < 0:
		age = "future"
	case duration < time.Minute:
		age = "now"
	case duration < time.Hour:
		age = fmt.Sprintf("%dm", int(duration.Minutes()))
	case duration < 24*time.Hour:
		age = fmt.Sprintf("%dh", int(duration.Hours()))
	case duration < 30*24*time.Hour:
		age = fmt.Sprintf("%dd", int(duration.Hours()/24))
	default:
		age = fmt.Sprintf("%dmo", int(duration.Hours()/(24*30)))
	}

	return age
}
