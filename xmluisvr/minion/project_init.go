package minion

import (
	"fmt"
	"io/fs"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

// InitProjectArgs configures project initialization
type InitProjectArgs struct {
	ProjectName string
	ConfigSlug  dt.PathSegment
	Writer      cliutil.Writer
}

// InitializeProjectConfig creates the .xmlui directory and config files
// This is independent of app file installation
func InitializeProjectConfig(args *InitProjectArgs) (err error) {
	var fp dt.Filepath
	var exists bool
	var configContent string

	if args == nil {
		err = fmt.Errorf("minion: InitProjectArgs is nil")
		goto end
	}

	// Note: The actual config creation happens via cfgstore.InitProjectConfig
	// which is external to this function. This function handles localsvr.json.

	// Write localsvr config
	fp, err = cfgstore.ProjectConfigFilepath(args.ConfigSlug, localsvr.ConfigFile)
	if err != nil {
		goto end
	}
	exists, err = fp.Exists()
	if err != nil {
		err = fmt.Errorf("checking file %s: %w", fp, err)
		goto end
	}

	if exists {
		goto end
	}

	configContent = generateLocalSvrConfig(args.ProjectName)
	err = fp.WriteFile([]byte(configContent), 0644)
	if err != nil {
		err = fmt.Errorf("writing file %s: %w", fp, err)
		goto end
	}

	args.Writer.V2().Printf("Creating localsvr.json config\n")

end:
	return err
}

// FileConflict represents a file that would be overwritten
type FileConflict struct {
	Path dt.RelFilepath
}

// CheckAppFileConflictsArgs configures file conflict checking
type CheckAppFileConflictsArgs struct {
	AppFS        fs.FS      // Embedded filesystem with app files
	AppSourceDir dt.DirPath // Directory within AppFS containing app files
}

// CheckAppFileConflicts checks if any files from the init-app would overwrite existing files
func CheckAppFileConflicts(args *CheckAppFileConflictsArgs) (conflicts []dt.RelFilepath, err error) {
	conflicts = make([]dt.RelFilepath, 0)

	if args == nil || args.AppFS == nil {
		err = fmt.Errorf("minion: CheckAppFileConflictsArgs is nil or missing AppFS")
		goto end
	}

	err = fs.WalkDir(args.AppFS, string(args.AppSourceDir), func(path string, d fs.DirEntry, err error) error {
		var fp dt.RelFilepath
		var exists bool

		if err != nil {
			goto end
		}
		if d.IsDir() {
			goto end
		}

		fp, err = dt.RelFilepath(path).Rel(args.AppSourceDir)
		if err != nil {
			goto end
		}

		exists, err = fp.Exists()
		if err != nil {
			goto end
		}

		if exists {
			conflicts = append(conflicts, fp)
		}

	end:
		return err
	})

end:
	return conflicts, err
}

// InitAppFilesArgs configures app file installation
type InitAppFilesArgs struct {
	AppFS           fs.FS      // Embedded filesystem with app files
	AppSourceDir    dt.DirPath // Directory within AppFS containing app files
	ShouldOverwrite bool       // Whether to overwrite existing files
	Writer          cliutil.Writer
}

// InitAppFiles writes the embedded app files to the current directory
func InitAppFiles(args *InitAppFilesArgs) (err error) {
	if args == nil || args.AppFS == nil {
		err = fmt.Errorf("minion: InitAppFilesArgs is nil or missing AppFS")
		goto end
	}

	err = fs.WalkDir(args.AppFS, string(args.AppSourceDir), func(path string, d fs.DirEntry, err error) error {
		var content []byte
		var relPath dt.RelFilepath

		if err != nil {
			goto end
		}
		if d.IsDir() {
			goto end
		}

		content, err = fs.ReadFile(args.AppFS, path)
		if err != nil {
			goto end
		}

		relPath, _ = dt.RelFilepath(path).Rel(args.AppSourceDir)
		err = relPath.WriteFile(content, 0644)
		if err != nil {
			goto end
		}
	end:
		return err
	})

	if err != nil {
		err = fmt.Errorf("writing app files: %w", err)
		goto end
	}

	args.Writer.Printf("Creating hello app files.\n")

end:
	return err
}

// FormatFileConflictError returns a formatted error message for file conflicts
func FormatFileConflictError(conflicts []dt.RelFilepath) string {
	msg := "\nCannot initialize app files because the following files already exist:\n"
	for _, file := range conflicts {
		msg += fmt.Sprintf("  - %s\n", file)
	}
	msg += "\nTo overwrite these files, use the --overwrite flag.\n"
	msg += "Alternatively, move or remove these files before running 'xmlui init --app'.\n"
	return msg
}

// generateLocalSvrConfig generates a minimal localsvr.json configuration using the localsvr API
func generateLocalSvrConfig(project string) (config string) {
	cfg := cfgldr.GenerateConfig(cfgldr.GenerateConfigArgs{
		Project: project,
	})
	config = cfg.String()
	return config
}
