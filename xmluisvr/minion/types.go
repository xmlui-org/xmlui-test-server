package minion

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
)

// SourceType represents the type of source for downloading demos
type SourceType string

const (
	UnspecifiedSourceType SourceType = ""
	GitHubSourceType      SourceType = "github" // GitHub archive (branch or tag)
	URLSourceType         SourceType = "url"    // URL to zip file
)

// ParseSourceType converts a string to a SourceType with validation
func ParseSourceType(s string) (st SourceType, err error) {
	st = SourceType(s)

	switch st {
	case UnspecifiedSourceType, GitHubSourceType, URLSourceType:
		// Valid type
	default:
		err = fmt.Errorf("invalid source type: %s", s)
	}

	return st, err
}

// DemoSource describes where to get demo content and where to install it
type DemoSource struct {
	Type       SourceType      // Type of source
	Repo       dt.URLSegments  // owner/repo format for GitHubSourceType
	URL        dt.URL          // URL for URLSourceType
	Ref        dt.Identifier   // Branch or tag
	Branch     dt.Identifier   // Branch or tag
	Tag        dt.Identifier   // Branch or tag
	Subdir     dt.PathSegments // optional subdirectory within source
	InstallDir dt.DirPath      // where to install (computed)
}

// Source is deprecated - use DemoSource instead
// Kept for backward compatibility during migration
type Source = DemoSource

// CopyRule defines a file copy operation with glob support
type CopyRule struct {
	From     string // glob pattern
	To       string // destination path (can be file or dir)
	Optional bool
}

// Variant represents an alternative configuration
type Variant struct {
	Slug dt.URLSegment
	Name string
	Copy []CopyRule
}
