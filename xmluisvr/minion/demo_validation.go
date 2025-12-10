package minion

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

// ValidateDemo performs comprehensive validation of a downloaded demo
// Returns error if the demo is not a valid XMLUI application
// Validation constants are typically defined in localsvr package
func ValidateDemo(webroot dt.DirPath, xmluiMarkers XMLUIMarkers, writer cliutil.Writer) (err error) {
	var indexPath dt.Filepath
	var mainPath dt.Filepath
	var exists bool
	var indexContent []byte
	var scriptSrc string
	var scriptPath dt.Filepath

	// 1. Check for index.html
	indexPath = dt.FilepathJoin(webroot, "index.html")
	exists, err = indexPath.Exists()
	if err != nil {
		err = fmt.Errorf("failed to check for index.html: %w", err)
		goto end
	}
	if !exists {
		err = fmt.Errorf("demo validation failed: missing index.html in %s", webroot)
		goto end
	}

	// 2. Check for Main.xmlui
	mainPath = dt.FilepathJoin(webroot, "Main.xmlui")
	exists, err = mainPath.Exists()
	if err != nil {
		err = fmt.Errorf("failed to check for Main.xmlui: %w", err)
		goto end
	}
	if !exists {
		err = fmt.Errorf("demo validation failed: missing Main.xmlui in %s", webroot)
		goto end
	}

	// 3. Parse index.html to find XMLUI script tag
	indexContent, err = indexPath.ReadFile()
	if err != nil {
		err = fmt.Errorf("failed to read index.html: %w", err)
		goto end
	}

	scriptSrc, err = extractScriptSrc(string(indexContent))
	if err != nil {
		goto end
	}
	if scriptSrc == "" {
		err = fmt.Errorf("demo validation failed: no script tag found in index.html")
		goto end
	}

	// 4. Validate script source
	if isFrontEndURL(scriptSrc) {
		// URL - we could optionally validate it's accessible, but skip for now
		// as it adds latency and network dependency
		writer.V2().Printf("  Script source: %s (external URL, skipping content validation)\n", scriptSrc)
		goto end
	}

	// Local file - check it exists and is valid XMLUI
	scriptPath = dt.FilepathJoin(webroot, scriptSrc)
	exists, err = scriptPath.Exists()
	if err != nil {
		err = fmt.Errorf("failed to check for script file: %w", err)
		goto end
	}
	if !exists {
		err = fmt.Errorf("demo validation failed: script file not found: %s", scriptPath)
		goto end
	}

	// 5. Validate it's a valid XMLUI JS file
	writer.V2().Printf("  Checking XMLUI script bundle: %s\n", scriptSrc)
	_, err = validateXMLUIScript(scriptPath, xmluiMarkers)
	if err != nil {
		err = fmt.Errorf("demo validation failed: invalid XMLUI bundle: %w", err)
		goto end
	}

end:
	return err
}

// DetectWebroot finds the directory containing index.html within the install directory
// It searches up to 2 levels deep to handle repos with subdirectories
func DetectWebroot(installDir dt.DirPath) (webroot dt.DirPath, err error) {
	var indexPath dt.Filepath
	var exists bool
	var entries []os.DirEntry
	var entry os.DirEntry
	var subdirPath dt.DirPath

	// Try the install directory itself first
	indexPath = dt.FilepathJoin(installDir, "index.html")
	exists, err = indexPath.Exists()
	if err != nil {
		err = fmt.Errorf("failed to check for index.html: %w", err)
		goto end
	}
	if exists {
		webroot = installDir
		goto end
	}

	// Try one level deep - look in subdirectories
	entries, err = installDir.ReadDir()
	if err != nil {
		err = fmt.Errorf("failed to read installation directory: %w", err)
		goto end
	}

	for _, entry = range entries {
		if !entry.IsDir() {
			continue
		}

		subdirPath = dt.DirPathJoin(installDir, entry.Name())
		indexPath = dt.FilepathJoin(subdirPath, "index.html")
		exists, err = indexPath.Exists()
		if err != nil {
			continue
		}
		if exists {
			webroot = subdirPath
			goto end
		}
	}

	// If we still haven't found index.html, fall back to installDir
	// (some demos might generate index.html dynamically)
	webroot = installDir

end:
	return webroot, err
}

// extractScriptSrc extracts the src attribute from the first script tag in HTML
// This is a simple implementation - could be enhanced with proper HTML parsing
func extractScriptSrc(htmlContent string) (src string, err error) {
	var scriptStart int
	var srcStart int
	var srcEnd int
	var quote byte
	var remainder string

	// Find <script tag
	scriptStart = strings.Index(htmlContent, "<script")
	if scriptStart == -1 {
		goto end
	}

	// Find src=" or src=' within the script tag
	remainder = htmlContent[scriptStart:]
	srcStart = strings.Index(remainder, "src=")
	if srcStart == -1 {
		goto end
	}

	// Determine quote type
	srcStart += 4 // Move past "src="
	if srcStart >= len(remainder) {
		goto end
	}

	quote = remainder[srcStart]
	if quote != '"' && quote != '\'' {
		goto end
	}

	srcStart++ // Move past opening quote
	srcEnd = strings.IndexByte(remainder[srcStart:], quote)
	if srcEnd == -1 {
		goto end
	}

	src = remainder[srcStart : srcStart+srcEnd]

end:
	return src, err
}

// validateXMLUIScript validates that a JavaScript file is a valid XMLUI bundle
// Uses contractual invariants as markers
func validateXMLUIScript(scriptPath dt.Filepath, markers XMLUIMarkers) (isValid bool, err error) {
	var content []byte
	var contentStr string
	var markerCount int
	var info os.FileInfo

	// Check file size
	info, err = scriptPath.Stat()
	if err != nil {
		err = fmt.Errorf("failed to stat script file: %w", err)
		goto end
	}

	if info.Size() < markers.MinSize {
		// Too small to be a valid XMLUI bundle
		goto end
	}

	// Read file content for marker detection
	content, err = readFileHead(scriptPath, markers.MarkerReadSize)
	if err != nil {
		err = fmt.Errorf("failed to read script file: %w", err)
		goto end
	}

	contentStr = string(content)

	// Check for markers
	markerCount = 0

	if strings.Contains(contentStr, markers.UMDExport) {
		markerCount++
	}

	if strings.Contains(contentStr, markers.CSSProps) {
		markerCount++
	}

	if strings.Contains(contentStr, markers.MarkupError) {
		markerCount++
	}

	if strings.Contains(contentStr, markers.FunctionLabel) {
		markerCount++
	}

	if strings.Contains(contentStr, markers.Version) {
		markerCount++
	}

	isValid = markerCount >= markers.MinMarkers

end:
	return isValid, err
}

// readFileHead reads the first n bytes of a file
func readFileHead(path dt.Filepath, maxBytes int) (content []byte, err error) {
	var file *os.File
	var n int

	file, err = path.Open()
	if err != nil {
		goto end
	}
	defer dt.CloseOrLog(file)

	content = make([]byte, maxBytes)
	n, err = file.Read(content)
	if err != nil && !errors.Is(err, io.EOF) {
		goto end
	}
	err = nil

	content = content[:n]

end:
	return content, err
}

// isFrontEndURL checks if a string is an HTTP or HTTPS URL
func isFrontEndURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// XMLUIMarkers contains the markers used to validate XMLUI bundles
type XMLUIMarkers struct {
	UMDExport      string
	CSSProps       string
	MarkupError    string
	FunctionLabel  string
	Version        string
	MinSize        int64
	MinMarkers     int
	MarkerReadSize int
}
