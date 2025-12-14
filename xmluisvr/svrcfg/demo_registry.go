package cfgldr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DemoRegistry represents the raw demo registry structure
type DemoRegistry struct {
	Schema      string `json:"$schema"`
	Version     int    `json:"version"`
	DefaultSlug string `json:"default_slug"`
	Demos       []Demo `json:"demos"`
}

// Demo represents a single demo entry in the registry
type Demo struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	ManifestURL string `json:"manifest_url"`
}

// LoadDemoRegistry reads the registry from the well-known location
// If the registry doesn't exist, it creates a default one.
func LoadDemoRegistry(configDir string) (registry *DemoRegistry, err error) {
	var registryPath string
	var data []byte

	registryPath = filepath.Join(configDir, "registry.json")

	// Check if registry exists
	_, err = os.Stat(registryPath)
	if os.IsNotExist(err) {
		// Create default registry
		registry = CreateDefaultRegistry()
		err = SaveRegistry(registryPath, registry)
		if err != nil {
			err = fmt.Errorf("failed to create default registry: %w", err)
			goto end
		}
		goto end
	}

	// Load existing registry
	data, err = os.ReadFile(registryPath)
	if err != nil {
		err = fmt.Errorf("failed to read registry: %w", err)
		goto end
	}

	registry = &DemoRegistry{}
	err = json.Unmarshal(data, registry)
	if err != nil {
		err = fmt.Errorf("failed to parse registry: %w", err)
		goto end
	}

end:
	return registry, err
}

// CreateDefaultRegistry creates the default registry with xmlui-invoice demo
func CreateDefaultRegistry() *DemoRegistry {
	return &DemoRegistry{
		Schema:      "https://xmlui.org/schemas/v1/demo-registry.json",
		Version:     1,
		DefaultSlug: "xmlui-invoice",
		Demos: []Demo{
			{
				Slug:        "xmlui-invoice",
				Name:        "XMLUI Invoice Demo",
				ManifestURL: "https://raw.githubusercontent.com/xmlui-org/xmlui-invoice/hajagosnorbert/demo/xmlui-manifest.json",
			},
		},
	}
}

// SaveRegistry writes the registry to disk
func SaveRegistry(path string, registry *DemoRegistry) (err error) {
	var data []byte

	// Ensure parent directory exists
	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		err = fmt.Errorf("failed to create registry directory: %w", err)
		goto end
	}

	data, err = json.MarshalIndent(registry, "", "  ")
	if err != nil {
		err = fmt.Errorf("failed to marshal registry: %w", err)
		goto end
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		err = fmt.Errorf("failed to write registry: %w", err)
		goto end
	}

end:
	return err
}
