package minion

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// zipExtensionRegex matches .zip extension case-insensitively
var zipExtensionRegex = regexp.MustCompile(`(?i)\.zip$`)

// httpURLRegex matches URLs starting with http:// or https://
var httpURLRegex = regexp.MustCompile(`^https?://`)

// ResolveDemoSourceArgs contains arguments for resolving a demo source
type ResolveDemoSourceArgs struct {
	SourceArg string         // User's source argument (repo name, org/repo, or URL)
	BranchArg string         // User's branch argument
	ConfigDir dt.DirPath     // Configuration directory for demo storage
	Reinstall bool           // --reinstall flag
	DryRun    bool           // --dry-run flag
	Writer    cliutil.Writer // For output messages
}

// ResolveDemoSource resolves user input to a DemoSource with computed InstallDir
func ResolveDemoSource(args *ResolveDemoSourceArgs) (ds *DemoSource, err error) {
	var sourceArg string
	var parts []string

	sourceArg = args.SourceArg

	ds = &DemoSource{
		Type: GitHubSourceType,
	}

	// Determine branch
	if args.BranchArg != "" {
		ds.Branch = dt.Identifier(args.BranchArg)
		ds.Ref = dt.Identifier(args.BranchArg)
	}

	// Case 1: No args or just "." — use default demo
	if sourceArg == "" || strings.TrimSpace(sourceArg) == "." {
		ds.Repo = dt.URLSegmentsJoin(localsvr.DefaultDemoOrg, localsvr.DefaultDemoRepo)
		goto end
	}

	if isHTTPURL(sourceArg) {
		// TODO Maybe throw an error here if ds.Branch != ""
		_, err = url.Parse(sourceArg)
		if err != nil {
			err = fmt.Errorf("invalid URL format: %s: %w", sourceArg, err)
			goto end
		}
		ds.InstallDir = urlToInstallPath(args.ConfigDir, sourceArg)
		ds.Type = URLSourceType
		ds.URL = dt.URL(sourceArg)
		goto end
	}

	// Case 2: org/repo or repo format
	parts = strings.Split(sourceArg, "/")
	switch len(parts) {
	case 2:
		ds.Repo = dt.URLSegmentsJoin(parts[0], parts[1])
	case 1:
		ds.Repo = dt.URLSegmentsJoin(localsvr.DefaultDemoOrg, parts[0])
	default:
		err = fmt.Errorf("invalid repository format: %s (expected 'repo' or 'org/repo')", sourceArg)
		goto end
	}

end:
	if err == nil && ds.InstallDir == "" && ds.Type == GitHubSourceType {
		// For GitHub sources, compute install path based on org/repo
		archiveURL := githubArchiveURL(ds.Repo, "main")
		ds.InstallDir = urlToInstallPath(args.ConfigDir, archiveURL)
	}

	return ds, err
}

// FindValidBranch finds an available branch and sets the install path
// This requires checking which branches exist on GitHub
func FindValidBranch(ds *DemoSource, configDir dt.DirPath, branches []string, writer cliutil.Writer) (result *DemoSource, err error) {
	var branch string
	var installPath dt.DirPath
	var installer *SiteInstaller
	var manifest *Manifest

	result = ds

	// If branch explicitly set, use it (already validated in ResolveDemoSource)
	if ds.Ref != "" || ds.Branch != "" {
		goto end
	}

	// Try each branch in order until one works
	for _, branch = range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		// Set install path using URL-based structure
		archiveURL := githubArchiveURL(ds.Repo, branch)
		installPath = urlToInstallPath(configDir, archiveURL)

		slug := ds.Repo.Base()

		// Build temporary manifest for validation
		manifest = &Manifest{
			Version:     1,
			Slug:        slug,
			Name:        fmt.Sprintf("XMLUI Demo: %s", slug),
			Description: fmt.Sprintf("Demo from %s @ %s", ds.Repo, branch),
			Source: DemoSource{
				Type: GitHubSourceType,
				Repo: ds.Repo,
				Ref:  dt.Identifier(branch),
			},
			Copy: []CopyRule{
				{
					From: "**",
					To:   ".",
				},
			},
		}

		// Create installer for validation
		installer = NewSiteInstaller(SiteInstallerArgs{
			Manifest:   manifest,
			ConfigDir:  configDir,
			InstallDir: installPath,
			SourceDir:  ".",
			Ref:        dt.Identifier(branch),
			DryRun:     false,
			Writer:     writer,
		})

		// Try to validate this branch
		writer.V2().Printf("Checking repository branch: %s\n", branch)
		err = installer.ValidateRepo()
		if err == nil {
			// Found a valid branch
			result = &DemoSource{
				Type:       ds.Type,
				Repo:       ds.Repo,
				Branch:     dt.Identifier(branch),
				Ref:        dt.Identifier(branch),
				URL:        ds.URL,
				InstallDir: installPath,
			}
			goto end
		}

		// Branch not found, try next
		writer.V2().Printf("  Branch '%s' not found, trying next...\n", branch)
	}

	// No valid branch found
	err = fmt.Errorf("no valid branch found (tried: %s)", strings.Join(branches, ", "))

end:
	return result, err
}

// isHTTPURL checks if a string is an HTTP or HTTPS URL
func isHTTPURL(s string) bool {
	return httpURLRegex.MatchString(s)
}

// githubArchiveURL constructs a GitHub archive URL for the given org/repo and ref
func githubArchiveURL(repo dt.URLSegments, ref string) string {
	return fmt.Sprintf("https://github.com/%s/archive/%s.zip", repo, ref)
}

// urlToInstallPath converts a URL to an install directory path
// For example: https://example.com/path/demo.zip -> .../demos/example.com/path/demo
func urlToInstallPath(configDir dt.DirPath, urlStr string) (installPath dt.DirPath) {
	var parsedURL *url.URL
	var pathWithoutScheme string

	parsedURL, _ = url.Parse(urlStr)

	// Build path as: host/path (without scheme)
	pathWithoutScheme = parsedURL.Host
	if parsedURL.Path != "" {
		pathWithoutScheme = pathWithoutScheme + parsedURL.Path
	}

	// Remove .zip extension if present (case-insensitive)
	pathWithoutScheme = zipExtensionRegex.ReplaceAllString(pathWithoutScheme, "")

	installPath = dt.DirPathJoin3(configDir, "demos", pathWithoutScheme)
	return installPath
}

// FormatDemoSource returns a human-readable description of the demo source
func FormatDemoSource(ds *DemoSource) (description string) {
	switch ds.Type {
	case GitHubSourceType:
		ref := ds.Ref
		if ref == "" {
			ref = ds.Branch
		}
		if ref == "" {
			ref = "main"
		}
		description = fmt.Sprintf("%s @ %s", ds.Repo, ref)
	case URLSourceType:
		description = string(ds.URL)
	default:
		description = "(unknown)"
	}
	return description
}

// urlToSlug extracts a reasonable slug from a URL
func urlToSlug(urlStr dt.URL) (slug dt.URLSegment) {
	var parsedURL *url.URL
	var parts []string

	parsedURL, _ = urlStr.Parse()

	// Try to extract meaningful name from path
	if parsedURL.Path != "" {
		parts = strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(parts) == 0 {
			goto end
		}
		slug = dt.URLSegment(parts[len(parts)-1])
		// Remove .zip extension case-insensitively
		slug = dt.URLSegment(zipExtensionRegex.ReplaceAllString(string(slug), ""))
		if slug != "" {
			goto end
		}
	}

	// Fallback to host if no useful path
	slug = dt.URLSegment(parsedURL.Host)
end:
	return slug
}
