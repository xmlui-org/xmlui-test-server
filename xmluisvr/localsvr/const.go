// Package localsvr provides shared utilities, types, and constants used throughout xmlui-localsvr.
//
// This package contains foundational utilities that are used across all other packages:
//
//   - Type definitions for file paths, identifiers, URLs, and other common concepts
//   - Parsing and validation functions for user input and configuration
//   - HTTP method and header utilities
//   - Database data type definitions and parsing
//   - Error definitions and handling utilities
//   - Logging setup and management
//   - File system operations and path manipulation
//
// # Core Types
//
// The package defines several string-based types for type safety:
//   - dt.Filepath: File system paths (absolute or relative)
//   - URLPath: HTTP URL paths with validation
//   - Identifier: Safe identifiers for variables and parameters
//   - ConnectString: Database connection strings
//   - HTTPMethod: HTTP request methods with validation
//
// # Parsing Functions
//
// Most types have corresponding ParseBytes* functions that validate input and return
// typed values or descriptive errors:
//
//	port, err := localsvr.ParseServerPort(8080, localsvr.ZeroInvalid)
//	method, err := localsvr.ParseHTTPMethod("GET", localsvr.EmptyInvalid)
//	path, err := localsvr.ParseURLPath("/api/v1/users")
//
// # Error Handling
//
// The package defines localsvr error types and provides utilities for error
// handling throughout the application, following Go best practices for
// error wrapping and context.
package localsvr

import (
	"path/filepath"

	"github.com/mikeschinkel/go-dt"
)

// Server Specific consts
const (
	// DefaultServerPort is the default HTTP port when none is specified.
	DefaultServerPort = 8080

	// LocalHostIP is the IP address for localhost connections.
	LocalHostIP = "127.0.0.1"

	// DefaultServerHost is the default host address for the HTTP server.
	DefaultServerHost = LocalHostIP
)

// AppInfo consts
const (
	AppVer dt.Version = "v0.0.0" // TODO To be changed soon

	// AppName is the human-readable name of the application.
	AppName                 = "XMLUI Local Server"
	AppDescr                = "Local development server for building and testing XMLUI web applications"
	AppSlug  dt.PathSegment = "xmlui-localsvr"

	// ConfigSlug provides the directory under ~/.config/ where configuration will be
	// stored. This is not xmlui-localsvr as everything XMLUI goes under the one location.
	ConfigSlug dt.PathSegment = "xmlui"

	// ConfigFile is the path for where the config file will be stored in the config
	// directory, e.g. ~/.config/xmlui/localsvr.json
	ConfigFile dt.RelFilepath = "localsvr.json"

	// ExeName is the standalone name for this app when compiled as a standalone.
	// HOWEVER, the `xmlui` CLI should really be the only executable we put on a
	// user's machine; everything else gets loaded by the one CLI executable. We
	// are merely enabling this app to be separately compiled into an executable
	// for our own convenince andwe do not expect to distribute it.
	ExeName dt.Filename = "xmlui-svr"

	LogPath dt.PathSegments = "logs"

	// GitHubRepoURL provides the GitHub repo for this project for use in error messages
	// TODO: Can we change this Github URL to be "https://github.com/xmlui-org/xmlui-test-server"?
	GitHubRepoURL = "https://github.com/xmlui-org/xmlui-test-server"
)

// Derived AppInfo consts
const (
	// InfoURL is Just a URL to display to users "for more information"
	InfoURL dt.URL = GitHubRepoURL

	LogFile dt.Filename = dt.Filename(string(AppSlug) + ".log")
)

var (
	ExtraInfo = map[string]any{
		"github_repo_url": GitHubRepoURL,
	}
)

const (
	// GitHubHostname is the hostname for GitHub repositories
	GitHubHostname dt.InternetDomain = "github.com"

	DemosPath         dt.PathSegment = "demos"
	WebrootPath       dt.PathSegment = "."
	DBRootPath        dt.PathSegment = "."
	DBFilename        dt.Filename    = "data.db"
	BootstrapFilename dt.Filename    = "bootstrap.sql"
)

// Demo defaults - used by demo command for resolving repositories and branches
const (
	DefaultDemoType     = "zip"
	DefaultDemoOrg      = "xmlui-org"
	DefaultDemoRepo     = "xmlui-todo"
	DefaultDemoBranches = "demo,main" // Comma-separated list of branches to try in order
)

const (
	ConfigPath     = "." + ConfigSlug
	ConfigFilename = dt.Filename(ConfigFile)
)

const UnknownVersion = "v0.0.0"

const (
	DefaultSQLite3ExtensionEntryPoint = "sqlite3_extension_init"
	DefaultOnFailurePolicy            = "warn"
	DefaultVarScope                   = "app"
	DefaultAPIBasePath                = "/api"
	DefaultAPIConfigFile              = "./api.json"
)

const (
	DefaultWebroot         = string(WebrootPath)
	DefaultDBRoot          = string(DBRootPath)
	DefaultSQLite3DBFile   = string(DBFilename)
	DefaultDBBootstrapFile = string(BootstrapFilename)
)

var (
	DefaultSQLite3Database     = filepath.Join(DefaultDBRoot, DefaultSQLite3DBFile)
	DefaultDBBootstrapFilepath = filepath.Join(DefaultDBRoot, DefaultDBBootstrapFile)
)

const (
	// XMLUIAppIndexFilename is the required index file for XMLUI apps
	XMLUIAppIndexFilename dt.Filename = "index.html"

	// XMLUIAppConfigFilename is the required config file for XMLUI apps
	XMLUIAppConfigFilename dt.Filename = "config.json"

	// XMLUIAppMainFilename is the main XMLUI markup file - strong indicator of an XMLUI app
	XMLUIAppMainFilename dt.Filename = "Main.xmlui"

	// XMLUIFileExtension is the file extension for XMLUI markup files
	XMLUIFileExtension dt.FileExt = ".xmlui"
)

// XMLUI bundle validation marker strings - contractual invariants in the XMLUI bundle
const (
	// XMLUIMarkerUMDExport is the global UMD export name pattern
	XMLUIMarkerUMDExport = ".xmlui="

	// XMLUIMarkerCSSProps is the XMLUI CSS custom properties prefix
	XMLUIMarkerCSSProps = "--xmlui-"

	// XMLUIMarkerMarkupError is the XMLUI markup validation error message
	XMLUIMarkerMarkupError = "Errors found while checking Xmlui markup"

	// XMLUIMarkerFunctionLabel is the diagnostic label for XMLUI functions
	XMLUIMarkerFunctionLabel = "[xmlui function]"

	// XMLUIMarkerVersion is the version logging string
	XMLUIMarkerVersion = "XMLUI version"
)

// XMLUI bundle validation size and count constants
const (
	// XMLUIBundleMinSize is the minimum file size for a valid XMLUI bundle (100KB)
	XMLUIBundleMinSize int64 = 100 * 1024

	// XMLUIBundleMarkerReadSize is the maximum bytes to read for marker detection (50KB)
	XMLUIBundleMarkerReadSize = 50 * 1024

	// XMLUIBundleMinMarkers is the minimum number of markers required to validate as XMLUI bundle
	XMLUIBundleMinMarkers = 2
)
