package minion

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// RefType defines the type of a ref (branch, tag, hash, or url)
type RefType string

const (
	BranchRefType RefType = "branch"
	TagRefType    RefType = "tag"
	HashRefType   RefType = "hash"
	URLRefType    RefType = "url"
)

// Demo holds information about an installed demo
type Demo struct {
	Domain      dt.InternetDomain `json:"domain"`               // "github.com", "example.com", etc.
	Org         dt.PathSegment    `json:"org,omitempty"`        // For GitHub: org name; For URLs: empty or host
	Repo        dt.PathSegment    `json:"repo,omitempty"`       // For GitHub: repo name; For URLs: last path segment
	Branch      dt.PathSegment    `json:"branch,omitempty"`     // Branch/ref name (for GitHub demos) - for improving UX
	Tag         dt.PathSegment    `json:"tag,omitempty"`        // Tag ref name (for GitHub demos) - for improving UX
	Hash        dt.PathSegment    `json:"hash,omitempty"`       // Hash ref value (for GitHub demos) - for improving UX
	Ref         dt.Identifier     `json:"ref,omitempty"`        // Internal ref value (branch/tag/hash name)
	RefType     RefType           `json:"type"`                 // Type of ref: branch, tag, hash, or url
	Path        dt.DirPath        `json:"path"`                 // Full path after host (for URL-based demos)
	InstallPath dt.DirPath        `json:"install_path"`         // Full filesystem path
	Description string            `json:"description"`          // From README
	Size        int64             `json:"-"`                    // Bytes. TODO show this in JSON when we calculate size
	Installed   time.Time         `json:"installed"`            // Last modified
	SourceURL   dt.URL            `json:"source_url,omitempty"` // ZIP URL used at install time
}

func (d *Demo) FullName() (name string) {
	switch d.RefType {
	case BranchRefType, TagRefType, HashRefType:
		name = fmt.Sprintf("%s#%s", dt.PathSegmentsJoin3(d.Domain, d.Org, d.Repo), d.Ref)
	case URLRefType:
		fallthrough
	default:
		name = filepath.Join(string(d.Domain), string(d.SourceURL))
	}
	return name
}

// JSON returns a JSON string representation of the Demo
func (d *Demo) JSON() (jsonText string) {
	var data []byte
	var err error

	if d == nil {
		jsonText = "null"
		goto end
	}

	data, err = json.MarshalIndent(d, "", "  ")
	if err != nil {
		jsonText = "{}"
		goto end
	}

	jsonText = string(data)

end:
	return jsonText
}

// Valid returns true if the demo has all required files
func (d *Demo) Valid() bool {
	return hasDemoFiles(d.InstallPath)
}

// ValidationErrors returns a list of validation error messages
func (d *Demo) ValidationErrors() []string {
	if d.Valid() {
		return []string{}
	}
	return collectValidationErrors(d.InstallPath)
}

// Demos is a slice of Demo pointers
type Demos []*Demo

// DemoSort defines how demos should be sorted
type DemoSort string

const (
	NameSort   DemoSort = "name"
	DateSort   DemoSort = "date"
	DomainSort DemoSort = "source"
)

// FullNames returns a slice of full names from the demo list
func (ds Demos) FullNames() []string {
	var names []string

	if len(ds) == 0 {
		goto end
	}

	names = make([]string, 0, len(ds))

	for _, d := range ds {
		if d == nil {
			continue
		}
		names = append(names, d.FullName())
	}

end:
	return names
}

// demoWithValidation wraps Demo and adds method-computed validation fields for JSON serialization
type demoWithValidation struct {
	*Demo            `json:",inline"`
	Valid            bool     `json:"valid"`
	ValidationErrors []string `json:"validation_errors"`
}

// JSON returns the list as a JSON string, including method-computed values
func (ds Demos) JSON() (jsonText string) {
	var data []byte
	var err error
	var enriched []demoWithValidation
	var d *Demo

	enriched = make([]demoWithValidation, 0, len(ds))

	for _, d = range ds {
		if d == nil {
			continue
		}

		enriched = append(enriched, demoWithValidation{
			Demo:             d,
			Valid:            d.Valid(),
			ValidationErrors: d.ValidationErrors(),
		})
	}

	data, err = json.MarshalIndent(enriched, "", "  ")
	if err != nil {
		jsonText = "[]"
		goto end
	}

	jsonText = string(data)

end:
	return jsonText
}

// DemoTableWriterArgs configures the table output
type DemoTableWriterArgs struct {
	// SortBy controls how rows are ordered. If empty, defaults to NameSort.
	SortBy DemoSort

	// SortDesc reverses the sort order when true.
	SortDesc bool

	// TimeFormat controls how Installed timestamps are rendered.
	// If empty, defaults to "2006-01-02".
	TimeFormat dt.TimeFormat

	// DemosDir, if non-empty, will be stripped as a prefix from InstallPath
	// for cleaner display. The header will indicate the stripped prefix.
	DemosDir dt.DirPath

	// HomeDir, if non-empty, will be replaced with "~" in the header display
	// for cleaner output.
	HomeDir dt.DirPath

	// ShowIndex enables row numbers via go-pretty's AutoIndex.
	ShowIndex bool

	// ShowSize enables the SIZE column displaying human-readable directory sizes.
	// (Deprecated: use Columns instead)
	ShowSize bool

	// Columns specifies which columns to display. If empty, defaults to DefaultColumns.
	Columns []DemoColumn

	// Style allows overriding the default table style (StyleLight).
	// If nil, StyleLight is used.
	Style *table.Style
}

// TableWriter returns a configured table.Writer for pretty printing the demo list
func (ds Demos) TableWriter(args DemoTableWriterArgs) (tw table.Writer) {
	var sortBy DemoSort
	var timeFormat dt.TimeFormat
	var rows Demos
	var columns []DemoColumn
	var ctx ColumnInfo
	var headerRow table.Row
	var colNum int
	var columnConfigs []table.ColumnConfig
	var d *Demo
	var row table.Row
	var col DemoColumn
	var meta *ColumnMeta
	var val string
	var newColumns []DemoColumn

	tw = table.NewWriter()
	columnIndex = 0 // Reset counter for this table

	sortBy = args.SortBy
	if sortBy == "" {
		sortBy = NameSort
	}

	timeFormat = args.TimeFormat
	if timeFormat == "" {
		timeFormat = time.DateOnly
	}

	// Determine columns to display
	columns = args.Columns
	if len(columns) == 0 {
		// Use default columns; if ShowSize is set, add SIZE to defaults
		columns = make([]DemoColumn, len(DefaultColumns))
		copy(columns, DefaultColumns)
		if args.ShowSize {
			// Insert SIZE before INSTALLED
			newColumns = nil
			for _, col = range columns {
				if col == DemoColumnInstalled {
					newColumns = append(newColumns, DemoColumnSize)
				}
				newColumns = append(newColumns, col)
			}
			columns = newColumns
		}
	}

	// Setup column context
	ctx = ColumnInfo{
		TimeNow:    time.Now(),
		TimeFormat: timeFormat,
		HomeDir:    args.HomeDir,
		DemosDir:   args.DemosDir,
	}

	if len(ds) > 0 {
		// Work on a copy so callers don't get surprising in-place reordering
		rows = make(Demos, len(ds))
		copy(rows, ds)
		sortDemos(rows, sortBy, args.SortDesc)

		// Build header row from columns
		headerRow = table.Row{}
		for _, col = range columns {
			meta = GetColumnMeta(col)
			if meta != nil {
				headerRow = append(headerRow, meta.Header)
			}
		}
		tw.AppendHeader(headerRow)

		// Build data rows
		for _, d = range rows {
			if d == nil {
				continue
			}

			row = table.Row{}
			for _, col = range columns {
				meta = GetColumnMeta(col)
				if meta != nil {
					val = meta.ValueFunc(d, &ctx)
					row = append(row, val)
				}
			}
			tw.AppendRow(row)
		}
	}

	// Configure column alignments from registry
	columnConfigs = []table.ColumnConfig{}
	colNum = 1
	for _, col = range columns {
		meta = GetColumnMeta(col)
		if meta != nil {
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Number: colNum,
				Align:  meta.Alignment,
			})
		}
		colNum++
	}
	tw.SetColumnConfigs(columnConfigs)

	// Apply style
	if args.Style != nil {
		tw.SetStyle(*args.Style)
	} else {
		tw.SetStyle(table.StyleLight)
	}
	tw.Style().Format.Header = text.FormatDefault

	return tw
}

// sortDemos sorts a Demos slice in place
func sortDemos(ds Demos, sortBy DemoSort, desc bool) {
	if sortBy == "" {
		sortBy = NameSort
	}

	sort.Slice(ds, func(i, j int) bool {
		di := ds[i]
		dj := ds[j]

		if di == nil || dj == nil {
			less := di != nil // non-nil first
			if desc {
				return !less
			}
			return less
		}

		var less bool

		switch sortBy {
		case DomainSort:
			if di.Domain != dj.Domain {
				less = di.Domain < dj.Domain
			} else {
				less = di.FullName() < dj.FullName()
			}

		case DateSort:
			// Default: newest first (After means di is newer)
			less = di.Installed.After(dj.Installed)

		default: // NameSort
			less = di.FullName() < dj.FullName()
		}

		if desc {
			return !less
		}
		return less
	})
}

// Logger interface for optional logging during demo discovery
type Logger interface {
	Warn(msg string, keyvals ...any)
}

// FindDemosArgs configures the FindDemos function
type FindDemosArgs struct {
	// ConfigDir is typically ~/.config/xmlui (or equivalent).
	// FindDemos will look inside ConfigDir/"demos".
	ConfigDir dt.DirPath

	// SortBy controls the sort order; defaults to NameSort if empty/unknown.
	SortBy DemoSort

	// SortDesc reverses the sort order when true.
	SortDesc bool

	// CalcSize enables directory size calculation for each demo.
	// This adds I/O overhead, so only enable when needed.
	CalcSize bool

	// Logger is optional. If nil, minion will not log warnings.
	Logger Logger
}

// FindDemos discovers installed demos and returns them as Demos
func FindDemos(args *FindDemosArgs) (demos Demos, err error) {
	var demosDir dt.DirPath
	var sources []fs.DirEntry
	var sortBy DemoSort

	if args == nil {
		err = fmt.Errorf("minion: FindDemosArgs is nil")
		goto end
	}

	demosDir = dt.DirPathJoin(args.ConfigDir, localsvr.DemosPath)

	sources, err = demosDir.ReadDir()
	if err != nil {
		if os.IsNotExist(err) {
			demos = Demos{}
			err = nil
			goto end
		}
		goto end
	}

	demos = Demos{}

	for _, sourceEntry := range sources {
		if !sourceEntry.IsDir() {
			continue
		}

		domain := dt.InternetDomain(sourceEntry.Name())
		sourceDir := dt.DirPathJoin(demosDir, domain)

		demos = append(
			demos,
			collectDemos(args.Logger, sourceDir, domain, args.CalcSize)...,
		)
	}

	// Normalize sort mode and sort.
	sortBy = args.SortBy
	if sortBy == "" {
		sortBy = NameSort
	}
	sortDemos(demos, sortBy, args.SortDesc)

end:
	return demos, err
}

// extractDescription extracts a description from demo directory README files.
// It looks for README.md, then README, then README.txt, and extracts the first
// line starting with '#'. If no README exists, it falls back to the directory name.
func extractDescription(demoDir dt.DirPath) (desc string) {
	var readmeFiles = []string{"README.md", "README", "README.txt"}
	var readmePath dt.Filepath
	var err error
	var exists bool
	var file *os.File
	var scanner *bufio.Scanner
	var line string
	var trimmed string

	for _, filename := range readmeFiles {
		readmePath = dt.FilepathJoin(demoDir, filename)
		exists, err = readmePath.Exists()
		if err != nil || !exists {
			continue
		}

		file, err = os.Open(string(readmePath))
		if err != nil {
			continue
		}

		scanner = bufio.NewScanner(file)
		for scanner.Scan() {
			line = scanner.Text()
			trimmed = strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				desc = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
				_ = file.Close()
				return desc
			}
		}
		_ = file.Close()
		// Found a README but no # line, keep looking
	}

	// Fallback: use directory base name
	desc = string(demoDir.Base())

	return desc
}

// collectDemos collects demos from a source directory using uniform URL-based paths.
// For GitHub demos (github.com/{org}/{repo}/archive/{ref}/), it also extracts
// org, repo, and branch metadata for display purposes.
func collectDemos(logger Logger, sourceDir dt.DirPath, domain dt.InternetDomain, calcSize bool) (demos Demos) {
	var de dt.DirEntry
	var err error
	var dirPath dt.DirPath
	var relPath dt.DirPath
	var stat os.FileInfo
	var demo *Demo

	// Walk the entire tree to find demo directories
	// Structure: demos/{host}/{path}/
	// GitHub: demos/github.com/{org}/{repo}/archive/{ref}/
	for de, err = range sourceDir.Walk() {
		switch {
		case err != nil:
			if logger != nil {
				logger.Warn("error walking demos", "source", domain, "error", err)
			}
			continue
		case de.Entry == nil:
			continue
		case !de.IsDir():
			continue
		}

		dirPath = de.DirPath()

		// Only collect directories that look like demo installations
		if !looksLikeDemoDir(dirPath) {
			continue
		}

		// Extract path relative to sourceDir
		relPath = dirPath.TrimPrefix(sourceDir).TrimPrefix("/")

		demo = &Demo{
			Domain:      domain,
			Path:        relPath,
			InstallPath: dirPath,
		}

		// For GitHub, parse out org/repo/ref from path structure
		if domain == localsvr.GitHubHostname {
			parseGitHubPath(demo, relPath)
		} else {
			// For non-GitHub demos, set RefType to URLRefType
			demo.RefType = URLRefType
		}

		// Extract description from README
		demo.Description = extractDescription(dirPath)

		// Get last modified time
		stat, err = dirPath.Stat()
		if err != nil {
			continue
		}
		demo.Installed = stat.ModTime()

		// Calculate size if requested
		if calcSize {
			demo.Size = calculateDirSize(dirPath)
		}

		demos = append(demos, demo)
		de.SkipDir() // Don't descend into demo directory
	}

	return demos
}

// parseGitHubPath extracts org, repo, and ref metadata from a GitHub demo path.
// Expected path format: {org}/{repo}/archive/{ref} or {org}/{repo}/refs/tags/{ref}
// Sets Org, Repo, Ref, RefType, and UX fields (Branch/Tag/Hash).
// TODO: This is violating the Parse*() pattern which is parse takes an input and returns an output and an error.
//
//	Need to fix this to follow the pattern. OR, this is NOT a parse but instead a Normalize*() method of Demo?
func parseGitHubPath(demo *Demo, relPath dt.DirPath) {
	var parts []string

	parts = strings.Split(string(relPath), "/")

	// Need at least 4 parts: org/repo/{archive|refs}/...
	if len(parts) < 4 {
		goto end
	}

	demo.Org = dt.PathSegment(parts[0])
	demo.Repo = dt.PathSegment(parts[1])

	if parts[2] == "archive" && len(parts) >= 4 {
		// Standard GitHub archive format: org/repo/archive/{ref}
		// Refs from archive are typically branches or tags
		demo.Ref = dt.Identifier(parts[3])

		// TODO: We need to discover what type is actually is and not just "default" to whatever is easy.
		demo.RefType = BranchRefType // Default to branch for archive
		demo.Branch = dt.PathSegment(parts[3])
		goto end
	}

	if parts[2] == "refs" && len(parts) >= 5 {
		// Explicit refs format: org/repo/refs/heads/{branch} or org/repo/refs/tags/{tag}
		demo.Ref = dt.Identifier(parts[4])
		if parts[3] == "heads" {
			demo.RefType = BranchRefType
			demo.Branch = dt.PathSegment(parts[4])
			demo.Ref = dt.Identifier(demo.Branch)
		} else if parts[3] == "tags" {
			demo.RefType = TagRefType
			demo.Tag = dt.PathSegment(parts[4])
			demo.Ref = dt.Identifier(demo.Tag)
		}
		goto end

	}

end:
	return
}

// hasDemoFiles checks if a directory is a demo directory
// A demo directory must contain both index.html and config.json
func hasDemoFiles(dir dt.DirPath) (hasFiles bool) {
	var exists bool
	var err error

	indexPath := dt.FilepathJoin(dir, localsvr.XMLUIAppIndexFilename)
	//configPath := dt.FilepathJoin(dir, localsvr.XMLUIAppConfigFilename)

	exists, err = indexPath.Exists()
	if err != nil || !exists {
		goto end
	}

	//exists, err = configPath.Exists()
	//if err != nil || !exists {
	//	goto end
	//}

	hasFiles = true

end:
	return hasFiles
}

// looksLikeDemoDir returns true if a directory appears to be a demo installation.
// Uses multiple signals to detect demo directories even if they're incomplete:
// - Strong signal: Main.xmlui file exists
// - Moderate signal: Any .xmlui file exists
// - Moderate signal: .xmlui config directory exists
func looksLikeDemoDir(dir dt.DirPath) (isDemo bool) {
	var exists bool
	var err error
	var entries []dt.DirEntry
	var entry dt.DirEntry
	var mainPath dt.Filepath
	var xmluiConfigDir dt.DirPath

	// Strong signal: Check for Main.xmlui
	mainPath = dt.FilepathJoin(dir, localsvr.XMLUIAppMainFilename)
	exists, err = mainPath.Exists()
	if err == nil && exists {
		isDemo = true
		goto end
	}

	// Moderate signal: Check for .xmlui config directory
	xmluiConfigDir = dt.DirPathJoin(dir, localsvr.ConfigPath)
	exists, err = xmluiConfigDir.Exists()
	if err == nil && exists {
		isDemo = true
		goto end
	}

	// Moderate signal: Check for any .xmlui file
	entries, err = dt.DirPathRead(dir)
	if err != nil {
		goto end
	}

	for _, entry = range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Ext() == localsvr.XMLUIFileExtension {
			isDemo = true
			goto end
		}
	}

end:
	return isDemo
}

// collectValidationErrors performs validation and returns error messages
func collectValidationErrors(dirPath dt.DirPath) []string {
	var errs []string
	var exists bool
	var err error

	// Check for required files
	indexPath := dt.FilepathJoin(dirPath, localsvr.XMLUIAppIndexFilename)
	exists, err = indexPath.Exists()
	if err != nil || !exists {
		errs = append(errs, "Missing index.html")
	}

	configPath := dt.FilepathJoin(dirPath, localsvr.XMLUIAppConfigFilename)
	exists, err = configPath.Exists()
	if err != nil || !exists {
		errs = append(errs, "Missing config.json")
	}

	// Could add more checks here (Main.xmlui, bundle validation, etc.)

	return errs
}

// formatSize returns a human-readable size string (e.g., "1.2 MB", "345 KB")
func formatSize(bytes int64) (size string) {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		size = fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		size = fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		size = fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		size = fmt.Sprintf("%d B", bytes)
	}

	return size
}

// calculateDirSize calculates the total size of all files in a directory tree
func calculateDirSize(dir dt.DirPath) (totalSize int64) {
	var de dt.DirEntry
	var err error

	for de, err = range dir.Walk() {
		if err != nil {
			continue
		}
		if de.Entry == nil || de.IsDir() {
			continue
		}

		info, err := de.Entry.Info()
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}

	return totalSize
}
