package minion

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtglob"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/cfgldr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// SiteInstaller handles the installation of a demo from a manifest
// TODO: Consider how this needs to evolve to support databases other an SQLite
type SiteInstaller struct {
	Manifest          *Manifest
	ConfigDir         dt.DirPath
	SourceDir         dt.DirPath
	Branch            string
	Tag               string
	Overwrite         bool
	DryRun            bool
	Writer            cliutil.Writer
	ConfigPath        dt.PathSegment
	ConfigFilename    dt.Filename
	DemosPath         dt.PathSegment
	WebrootPath       dt.PathSegment
	DBRootPath        dt.PathSegment
	DBFilename        dt.Filename
	BootstrapFilename dt.Filename
}

type SiteInstallerArgs struct {
	Manifest  *Manifest
	ConfigDir dt.DirPath
	SourceDir dt.DirPath
	Branch    string
	Tag       string
	Overwrite bool
	DryRun    bool
	Writer    cliutil.Writer
}

func NewSiteInstaller(args SiteInstallerArgs) *SiteInstaller {
	return &SiteInstaller{
		ConfigPath:        localsvr.ConfigPath,
		ConfigFilename:    localsvr.ConfigFilename,
		DemosPath:         localsvr.DemosPath,
		WebrootPath:       localsvr.WebrootPath,
		DBRootPath:        localsvr.DBRootPath,
		DBFilename:        localsvr.DBFilename,
		BootstrapFilename: localsvr.BootstrapFilename,
		Manifest:          args.Manifest,
		ConfigDir:         args.ConfigDir,
		SourceDir:         args.SourceDir,
		Branch:            args.Branch,
		Tag:               args.Tag,
		Overwrite:         args.Overwrite,
		DryRun:            args.DryRun,
		Writer:            args.Writer,
	}
}

type InstallResult struct {
	InstallDir dt.DirPath
	ConfigFile dt.Filepath
}

func (si *SiteInstaller) installDir() dt.DirPath {
	return dt.DirPathJoin3(si.ConfigDir, si.DemosPath, si.Manifest.Slug)
}

func (si *SiteInstaller) configFile() dt.Filepath {
	return dt.FilepathJoin3(si.installDir(), si.ConfigPath, si.ConfigFilename)
}

// Install downloads, copies, and configures a demo in the config directory
func (si *SiteInstaller) Install() (result *InstallResult, err error) {
	var sourceDir dt.DirPath
	var globRules *dtglob.GlobRules
	var job *CopyJob
	var exists bool

	installDir := si.installDir()

	if si.DryRun {
		// In dry-run mode, use a descriptive placeholder path
		//sourceDir = dt.DirPath(fmt.Sprintf("<temp-download>/%s", si.Manifest.Slug))
		si.Writer.Printf("Downloading source from %s...\n", si.Manifest.Source.Repo)
		si.Writer.Printf("Installing demo files...\n")
		si.Writer.Printf("✓ Dry run complete (no changes made)\n")
		goto end
	}

	// Step 1: Check if demo already exists
	exists, err = installDir.Exists()
	if err != nil {
		err = fmt.Errorf("failed to check existence of install directory %s: %w", installDir, err)
		goto end
	}
	if exists && !si.Overwrite {
		err = fmt.Errorf("demo '%s' already exists at %s\nUse --overwrite to download again", si.Manifest.Slug, installDir)
		goto end
	}

	// Step 2: Download source
	si.Writer.Printf("Downloading source from %s...\n", si.Manifest.Source.Repo)
	sourceDir, err = si.Download()
	if err != nil {
		err = fmt.Errorf("failed to download source: %w", err)
		goto end
	}
	defer cleanupTempDir(sourceDir)

	// Step 3: Copy demo files
	si.Writer.Printf("Installing demo files...\n")

	// Convert runpkg.CopyRule to dtglob.GlobRules
	globRules, err = ParseGlobRules(si.Manifest.Copy, sourceDir)
	if err != nil {
		err = fmt.Errorf("failed to parse copy rules: %w", err)
		goto end
	}

	// Create and execute copy job
	job = NewCopyJob(CopyJobArgs{
		GlobRules: globRules,
		DestDir:   installDir,
		Overwrite: si.Overwrite,
		DryRun:    si.DryRun,
		Writer:    si.Writer,
	})

	err = job.Run()
	if err != nil {
		err = fmt.Errorf("failed to copy files: %w", err)
		goto end
	}

	// Step 4: Ensure config exists
	err = si.ensureConfig(installDir)
	if err != nil {
		err = fmt.Errorf("failed to ensure config: %w", err)
		goto end
	}

	si.Writer.Printf("✓ Demo installed successfully\n")

end:
	if err == nil {
		result = &InstallResult{
			InstallDir: si.installDir(),
			ConfigFile: si.configFile(),
		}
	}
	return result, err
}

// ensureConfig ensures a localsvr.json config file exists in the install directory
func (si *SiteInstaller) ensureConfig(installDir dt.DirPath) (err error) {
	var configDir dt.DirPath
	var configFile dt.Filepath
	var exists bool

	// Config goes in <installDir>/.xmlui/localsvr.json
	configDir = dt.DirPathJoin(installDir, si.ConfigPath)
	configFile = dt.FilepathJoin(configDir, si.ConfigFilename)

	// Check if config already exists
	exists, _ = configFile.Exists()
	err = nil

	if exists {
		si.Writer.Printf("Using existing config: %s\n", configFile)
		goto end
	}

	// Generate new config
	si.Writer.Printf("Generating config: %s\n", configFile)
	err = si.generateConfig(installDir, configFile)

end:
	return err
}

// generateConfig creates a new localsvr.json config file for the demo
func (si *SiteInstaller) generateConfig(installDir dt.DirPath, configFile dt.Filepath) (err error) {
	var cfg *cfgldr.RootConfigV1
	var logsDir dt.DirPath
	var dbPath dt.Filepath
	var bootstrapPath dt.Filepath

	// Paths relative to install directory
	webrootPath := string(dt.DirPathJoin(installDir, si.WebrootPath))
	dbPath = dt.FilepathJoin3(installDir, si.DBRootPath, si.DBFilename)
	bootstrapPath = dt.FilepathJoin3(installDir, si.DBRootPath, si.BootstrapFilename)

	// Generate config using the public cfgldr function
	cfg = cfgldr.GenerateConfig(cfgldr.GenerateConfigArgs{
		Webroot:     webrootPath,
		DBPath:      string(dbPath),
		DBBootstrap: string(bootstrapPath),
	})

	// Ensure config directory exists
	err = configFile.Dir().MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create config directory: %w", err)
		goto end
	}

	// Ensure logs directory exists
	logsDir = dt.DirPathJoin(installDir, "logs")
	err = logsDir.MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create logs directory: %w", err)
		goto end
	}

	// Write config file
	err = configFile.WriteFile(cfg.Bytes(), 0644)
	if err != nil {
		err = fmt.Errorf("failed to write config file: %w", err)
		goto end
	}

end:
	return err
}

// cleanupTempDir removes the temporary directory created during download
func cleanupTempDir(tempDir dt.DirPath) {
	var parent dt.DirPath

	// Extract the actual temp directory from the content path
	// tempDir might be like /tmp/xmlui-download-xxx/extracted/repo-name
	// We want to remove the /tmp/xmlui-download-xxx part
	for tempDir.Contains(DownloadDirPrefix) {
		parent = tempDir.Dir()
		if parent.Contains(DownloadDirPrefix) {
			tempDir = parent
			continue
		}
		if parent.Base().Contains(DownloadDirPrefix) {
			tempDir = parent
			goto end
		}
		goto end
	}
end:
	dt.LogOnError(tempDir.RemoveAll())
}

// Download downloads the source content based on the manifest
// Returns the path to the extracted content
func (si *SiteInstaller) Download() (contentDir dt.DirPath, err error) {
	var downloadURL dt.URL
	var tempDir dt.DirPath
	var zipPath dt.Filepath
	var extractDir dt.DirPath

	// Build the download URL based on source type
	downloadURL, err = si.downloadURL()
	if err != nil {
		goto end
	}
	// Create temp directory for download
	tempDir, err = dt.MkdirTemp("", DownloadDirPrefix+"*")
	if err != nil {
		err = fmt.Errorf("failed to create temp directory: %w", err)
		goto end
	}

	// Download the archive
	zipPath = dt.FilepathJoin(tempDir, "source.zip")
	err = downloadFile(downloadURL, zipPath)
	if err != nil {
		err = fmt.Errorf("failed to download source: %w", err)
		goto end
	}

	// Extract the archive
	extractDir = dt.DirPathJoin(tempDir, "extracted")
	err = extractDir.MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create extraction directory: %w", err)
		goto end
	}

	err = unzipFile(zipPath, extractDir)
	if err != nil {
		err = fmt.Errorf("failed to extract archive: %w", err)
		goto end
	}

	// GitHub archives contain a single top-level directory
	// Find it and return its path
	contentDir, err = findTopLevelDir(extractDir)
	if err != nil {
		goto end
	}

	// If subdir is specified in manifest, navigate to it
	if si.Manifest.Source.Subdir != "" && si.Manifest.Source.Subdir != "." {
		contentDir = dt.DirPathJoin(contentDir, si.Manifest.Source.Subdir)
		_, err = contentDir.Stat()
		if err != nil {
			err = fmt.Errorf("subdir '%s' not found in source", si.Manifest.Source.Subdir)
			goto end
		}
	}

end:
	if err != nil && tempDir != "" {
		dt.LogOnError(tempDir.RemoveAll())
	}
	return contentDir, err

}

func (si *SiteInstaller) downloadURL() (url dt.URL, err error) {
	var repo string
	var ref string
	var source = si.Manifest.Source
	repo = source.Repo

	// Determine the ref (branch or tag)
	switch {
	case si.Tag != "":
		ref = si.Tag
	case si.Branch != "":
		ref = si.Branch
	case source.Tag != "":
		ref = source.Tag
	case source.Branch != "":
		ref = source.Branch
	default:
		err = fmt.Errorf("no branch or tag specified in manifest or flags")
		goto end
	}

	switch source.Type {
	case "zip":
		// GitHub archive URL for branch or tag
		url = dt.URL(fmt.Sprintf("https://github.com/%s/archive/refs/heads/%s.zip", repo, ref))

	case "release":
		// GitHub release asset URL (tag-based)
		url = dt.URL(fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.zip", repo, ref))

	default:
		err = fmt.Errorf("unsupported source type: %s", source.Type)
		goto end
	}

end:
	return url, err
}

// ValidateRepo checks if the repository archive exists using HTTP HEAD
func (si *SiteInstaller) ValidateRepo() (err error) {
	var url dt.URL
	var client *http.Client
	var resp *http.Response
	var repoURL string

	url, err = si.downloadURL()
	if err != nil {
		goto end
	}

	// Build user-friendly GitHub repo URL
	repoURL = fmt.Sprintf("https://github.com/%s", si.Manifest.Source.Repo)

	client = &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err = client.Head(string(url))
	if err != nil {
		err = fmt.Errorf("failed to check repository: %w", err)
		goto end
	}
	defer dt.CloseOrLog(resp.Body)

	if resp.StatusCode == 404 {
		err = fmt.Errorf("repository not found: %s", repoURL)
		goto end
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("unexpected status code %d when checking repository: %s", resp.StatusCode, repoURL)
		goto end
	}

end:
	return err
}

// downloadFile downloads a file from URL to destination path
func downloadFile(url dt.URL, destPath dt.Filepath) (err error) {
	var client *http.Client
	var resp *http.Response
	var out *os.File

	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err = url.GET(client)
	if err != nil {
		err = fmt.Errorf("failed to download: %w", err)
		goto end
	}
	defer dt.CloseOrLog(resp.Body)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
		goto end
	}

	out, err = destPath.Create()
	if err != nil {
		err = fmt.Errorf("failed to create file: %w", err)
		goto end
	}
	defer dt.CloseOrLog(out)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to write file: %w", err)
		goto end
	}

end:
	return err
}

// unzipFile extracts a zip archive to the destination directory
func unzipFile(zipPath dt.Filepath, destDir dt.DirPath) (err error) {
	var r *zip.ReadCloser
	var zf *zip.File
	var fp dt.Filepath
	var cleanDest dt.DirPath
	var errs []error

	r, err = zip.OpenReader(string(zipPath))
	if err != nil {
		err = fmt.Errorf("failed to open zip: %w", err)
		goto end
	}
	defer dt.CloseOrLog(r)

	cleanDest = destDir.Clean().EnsureTrailSep()
	for _, zf = range r.File {
		fp = dt.FilepathJoin(destDir, zf.Name)

		// Check for ZipSlip vulnerability
		if !fp.HasPrefix(cleanDest) {
			err = fmt.Errorf("illegal file path (ZipSlip protection): %s", fp)
			errs = dt.AppendErr(errs, err)
			continue
		}

		if zf.FileInfo().IsDir() {
			err = dt.DirPath(fp).MkdirAll(os.ModePerm)
			errs = dt.AppendErr(errs, err)
			continue
		}

		// Create parent directory if needed
		err = fp.Dir().MkdirAll(os.ModePerm)
		if err != nil {
			errs = dt.AppendErr(errs, err)
			continue
		}

		err = writeFileFromZip(zf, fp)
		if err != nil {
			errs = dt.AppendErr(errs, err)
			continue
		}
	}

end:
	return err
}

func writeFileFromZip(zf *zip.File, fp dt.Filepath) (err error) {
	var outFile *os.File
	var rc io.ReadCloser

	// Extract file
	outFile, err = fp.OpenFile(os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zf.Mode())
	if err != nil {
		goto end
	}
	defer dt.CloseOrLog(outFile)

	rc, err = zf.Open()
	if err != nil {
		goto end
	}
	defer dt.CloseOrLog(rc)

	_, err = io.Copy(outFile, rc)
	if err != nil {
		goto end
	}
end:
	return err
}

// findTopLevelDir finds the top-level directory in an extracted archive
func findTopLevelDir(extractDir dt.DirPath) (topLevelDir dt.DirPath, err error) {
	var entries []os.DirEntry
	var entry os.DirEntry

	entries, err = extractDir.ReadDir()
	if err != nil {
		err = fmt.Errorf("failed to read extracted directory: %w", err)
		goto end
	}

	if len(entries) == 0 {
		err = fmt.Errorf("extracted archive is empty")
		goto end
	}

	// Find the first directory entry
	for _, entry = range entries {
		if entry.IsDir() {
			topLevelDir = dt.DirPathJoin(extractDir, entry.Name())
			goto end
		}
	}

	// If no directory found, return the extract dir itself
	topLevelDir = extractDir

end:
	return topLevelDir, err
}
