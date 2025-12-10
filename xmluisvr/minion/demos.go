package minion

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// Demo holds information about an installed demo
type Demo struct {
	Domain      dt.InternetDomain `json:"domain"`      // "github.com", "example.com", etc.
	Org         dt.PathSegment    `json:"org"`         // For GitHub: org name; For URLs: empty or host
	Repo        dt.PathSegment    `json:"repo"`        // For GitHub: repo name; For URLs: last path segment
	Branch      dt.PathSegment    `json:"branch"`      // Branch/ref name (for GitHub demos)
	Path        dt.DirPath        `json:"path"`        // Full path after host (for URL-based demos)
	FullName    dt.PathSegments   `json:"fullName"`    // Display name: "org/repo" or "host/path"
	InstallPath dt.DirPath        `json:"installPath"` // Full filesystem path
	Description string            `json:"description"` // From README
	Size        int64             `json:"size"`        // Bytes
	Installed   time.Time         `json:"installed"`   // Last modified
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
func (ds Demos) FullNames() []dt.PathSegments {
	var names []dt.PathSegments

	if len(ds) == 0 {
		goto end
	}

	names = make([]dt.PathSegments, 0, len(ds))

	for _, d := range ds {
		if d == nil {
			continue
		}
		names = append(names, d.FullName)
	}

end:
	return names
}

// JSON returns the list as a JSON string
func (ds Demos) JSON() (jsonText string) {
	var data []byte
	var err error

	data, err = json.MarshalIndent(ds, "", "  ")
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
	ShowSize bool

	// Style allows overriding the default table style (StyleLight).
	// If nil, StyleLight is used.
	Style *table.Style
}

// TableWriter returns a configured table.Writer for pretty printing the demo list
func (ds Demos) TableWriter(args DemoTableWriterArgs) (tw table.Writer) {
	var sortBy DemoSort
	var timeFormat dt.TimeFormat
	var demosDir dt.DirPath
	var homeDir dt.DirPath
	var rows Demos
	var pathHeader string
	var headerPath dt.DirPath
	var columnConfigs []table.ColumnConfig
	var colNum int

	tw = table.NewWriter()

	if len(ds) == 0 {
		goto configure
	}

	sortBy = args.SortBy
	if sortBy == "" {
		sortBy = NameSort
	}

	timeFormat = args.TimeFormat
	if timeFormat == "" {
		timeFormat = "2006-01-02"
	}

	demosDir = args.DemosDir
	if demosDir != "" && !demosDir.HasSuffix("/") {
		demosDir += "/"
	}

	homeDir = args.HomeDir

	// Work on a copy so callers don't get surprising in-place reordering.
	rows = make(Demos, len(ds))
	copy(rows, ds)
	sortDemos(rows, sortBy, args.SortDesc)

	// Header - indicate the stripped prefix if applicable
	if demosDir != "" {
		headerPath = demosDir.TrimSuffix("/")
		// Replace home directory with ~ for cleaner display
		if homeDir != "" && headerPath.HasPrefix(homeDir) {
			headerPath = "~" + headerPath[len(homeDir):]
		}
		pathHeader = fmt.Sprintf("PATH (within %s/)", headerPath)
	} else {
		pathHeader = "PATH"
	}

	// Build header row based on options
	if args.ShowSize {
		tw.AppendHeader(table.Row{"DEMO", pathHeader, "SIZE", "LAST MODIFIED"})
	} else {
		tw.AppendHeader(table.Row{"DEMO", pathHeader, "LAST MODIFIED"})
	}

	for _, d := range rows {
		var path dt.DirPath
		var date string
		var row table.Row

		if d == nil {
			continue
		}

		path = d.InstallPath
		if demosDir != "" && d.InstallPath.HasPrefix(demosDir) {
			path = path[len(demosDir):]
		}

		if !d.Installed.IsZero() {
			date = d.Installed.Format(string(timeFormat))
		}

		row = table.Row{d.FullName, path}
		if args.ShowSize {
			row = append(row, formatSize(d.Size))
		}
		row = append(row, date)
		tw.AppendRow(row)
	}

configure:
	// Enable auto-index if requested
	if args.ShowIndex {
		tw.SetAutoIndex(true)
	}

	// Configure column alignments
	colNum = 1
	columnConfigs = []table.ColumnConfig{
		{Number: colNum, Align: text.AlignLeft}, // DEMO
	}
	colNum++
	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Number: colNum, Align: text.AlignLeft, // PATH
	})
	colNum++
	if args.ShowSize {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Number: colNum, Align: text.AlignRight, // SIZE
		})
		colNum++
	}
	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Number: colNum, Align: text.AlignRight, // LAST MODIFIED
	})
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
				less = di.FullName < dj.FullName
			}

		case DateSort:
			// Default: newest first (After means di is newer)
			less = di.Installed.After(dj.Installed)

		default: // NameSort
			less = di.FullName < dj.FullName
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

// ListDemosArgs configures the ListDemos function
type ListDemosArgs struct {
	// ConfigDir is typically ~/.config/xmlui (or equivalent).
	// ListDemos will look inside ConfigDir/"demos".
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

// ListDemos discovers installed demos and returns them as Demos
func ListDemos(args *ListDemosArgs) (demos Demos, err error) {
	var demosDir dt.DirPath
	var sources []fs.DirEntry
	var sortBy DemoSort

	if args == nil {
		err = fmt.Errorf("minion: ListDemosArgs is nil")
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

		// Check if this is a demo directory (contains index.html and config.json)
		if !hasDemoFiles(dirPath) {
			continue
		}

		// Extract path relative to sourceDir
		relPath = dirPath.TrimPrefix(sourceDir).TrimPrefix("/")

		demo = &Demo{
			Domain:      domain,
			Path:        relPath,
			InstallPath: dirPath,
		}

		// For GitHub, parse out org/repo/branch from path structure
		// Path format: {org}/{repo}/archive/{ref}
		if domain == localsvr.GitHubHostname {
			parseGitHubPath(demo, relPath)
		} else {
			demo.FullName = dt.PathSegmentsJoin(domain, relPath)
		}

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

// parseGitHubPath extracts org, repo, and branch from a GitHub demo path.
// Expected path format: {org}/{repo}/archive/{ref}
func parseGitHubPath(demo *Demo, relPath dt.DirPath) {
	parts := strings.Split(string(relPath), "/")

	// Need at least 4 parts: org/repo/archive/ref
	if len(parts) >= 4 && parts[2] == "archive" {
		demo.Org = dt.PathSegment(parts[0])
		demo.Repo = dt.PathSegment(parts[1])
		demo.Branch = dt.PathSegment(parts[3])
		demo.FullName = dt.PathSegmentsJoin3(demo.Domain, demo.Org, demo.Repo)
	} else {
		// Fallback for non-standard GitHub paths
		demo.FullName = dt.PathSegmentsJoin(demo.Domain, relPath)
	}
}

// hasDemoFiles checks if a directory is a demo directory
// A demo directory must contain both index.html and config.json
func hasDemoFiles(dir dt.DirPath) (hasFiles bool) {
	var exists bool
	var err error

	indexPath := dt.FilepathJoin(dir, localsvr.XMLUIAppIndexFilename)
	configPath := dt.FilepathJoin(dir, localsvr.XMLUIAppConfigFilename)

	exists, err = indexPath.Exists()
	if err != nil || !exists {
		goto end
	}

	exists, err = configPath.Exists()
	if err != nil || !exists {
		goto end
	}

	hasFiles = true

end:
	return hasFiles
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
