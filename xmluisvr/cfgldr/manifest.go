package cfgldr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/mikeschinkel/go-dt"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Manifest represents the raw xmlui-manifest.json structure
type Manifest struct {
	Schema      string     `json:"$schema"`
	Version     int        `json:"version"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Source      Source     `json:"source"`
	Copy        []CopyRule `json:"copy"`
	Variants    []Variant  `json:"variants,omitempty"`
}

// Source describes where to get the demo/project content
type Source struct {
	Type   string `json:"type"`             // zip | release | clone | local
	Repo   string `json:"repo"`             // owner/repo format
	Branch string `json:"branch,omitempty"` // mutually exclusive with Tag
	Tag    string `json:"tag,omitempty"`    // mutually exclusive with Branch
	Subdir string `json:"subdir,omitempty"` // optional subdirectory within source
}

// CopyRule defines a file copy operation with glob support
type CopyRule struct {
	From     string `json:"from"` // glob pattern
	To       string `json:"to"`   // destination path (can be file or dir)
	Optional bool   `json:"optional,omitempty"`
}

// Variant represents an alternative configuration
type Variant struct {
	Slug string     `json:"slug"`
	Name string     `json:"name"`
	Copy []CopyRule `json:"copy"`
}

var httpRegex = regexp.MustCompile("^https?://")

// LoadManifest loads and parses a manifest from the given URL or file path
func LoadManifest(manifestURLOrPath string) (manifest *Manifest, err error) {
	var resp *http.Response
	var body []byte

	// Check if it's a URL (http/https) or a local file path
	//goland:noinspection HttpUrlsUsage
	if httpRegex.MatchString(manifestURLOrPath) {
		// Load from URL
		resp, err = httpClient.Get(manifestURLOrPath)
		if err != nil {
			err = fmt.Errorf("failed to fetch manifest: %w", err)
			goto end
		}
		defer dt.CloseOrLog(resp.Body)

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
			goto end
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed to read manifest response: %w", err)
			goto end
		}
	} else {
		// Load from local file
		body, err = os.ReadFile(manifestURLOrPath)
		if err != nil {
			err = fmt.Errorf("failed to read manifest file: %w", err)
			goto end
		}
	}

	manifest = &Manifest{}
	err = json.Unmarshal(body, manifest)
	if err != nil {
		err = fmt.Errorf("failed to parse manifest JSON: %w", err)
		goto end
	}

end:
	return manifest, err
}
