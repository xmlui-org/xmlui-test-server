package minion

import (
	"fmt"

	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

// FetchManifest downloads, parses, and type-checks a manifest from the given URL
func FetchManifest(manifestURL string) (manifest *Manifest, err error) {
	var rawManifest *cfgldr.Manifest

	// Load raw JSON using clicfg
	rawManifest, err = cfgldr.LoadManifest(manifestURL)
	if err != nil {
		goto end
	}

	// Parse into type-checked runpkg.Manifest
	manifest, err = ParseManifest(rawManifest)
	if err != nil {
		err = fmt.Errorf("failed to parse manifest: %w", err)
		goto end
	}

end:
	return manifest, err
}
