package minion

import (
	"github.com/mikeschinkel/go-dt"
)

// Manifest represents the type-checked xmlui-manifest.json structure
type Manifest struct {
	Schema      dt.URL
	Version     int
	Slug        dt.URLSegment
	Name        string
	Description string
	Source      DemoSource
	Copy        []CopyRule
	Variants    []Variant
}
