package minion

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtglob"
)

// CopyJob encapsulates a complete file copy operation based on manifest rules
type CopyJob struct {
	GlobRules *dtglob.GlobRules // What files to copy (pattern-based rules)
	DestDir   dt.DirPath        // Where to install
	Overwrite bool              // Overwrite existing files
	ModeFunc  dt.EntryModeFunc  // Permission callback (nil = preserve source)
	DryRun    bool              // Simulate without changes
	Writer    cliutil.Writer    // Progress/status output
}

// CopyJobArgs provides named parameters for construction
type CopyJobArgs struct {
	GlobRules *dtglob.GlobRules
	DestDir   dt.DirPath
	Overwrite bool
	ModeFunc  dt.EntryModeFunc
	DryRun    bool
	Writer    cliutil.Writer
}

// NewCopyJob creates a new CopyJob with the given parameters
func NewCopyJob(args CopyJobArgs) *CopyJob {
	return &CopyJob{
		GlobRules: args.GlobRules,
		DestDir:   args.DestDir,
		Overwrite: args.Overwrite,
		ModeFunc:  args.ModeFunc,
		DryRun:    args.DryRun,
		Writer:    args.Writer,
	}
}

// Run executes the copy job
func (cj *CopyJob) Run() (err error) {
	if cj.DryRun {
		err = cj.dryRun()
		goto end
	}

	err = cj.GlobRules.CopyTo(cj.DestDir, &dt.CopyOptions{
		Overwrite:    cj.Overwrite,
		DestModeFunc: cj.ModeFunc,
	})

end:
	return err
}

// dryRun simulates the copy operation without making changes
func (cj *CopyJob) dryRun() (err error) {
	cj.Writer.Printf("[DRY RUN] Would copy files from %s to %s\n",
		cj.GlobRules.BaseDir, cj.DestDir)
	// TODO: Enumerate what would be copied
	return err
}
