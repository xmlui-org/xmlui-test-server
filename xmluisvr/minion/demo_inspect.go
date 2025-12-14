package minion

import (
	"fmt"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

// DemoSelectionMode defines how to select/filter demos
type DemoSelectionMode string

const (
	SelectAll        DemoSelectionMode = "all"
	SelectByName     DemoSelectionMode = "by_name"
	SelectWithErrors DemoSelectionMode = "with_errors"
)

// SelectDemosArgs configures the demo selection
type SelectDemosArgs struct {
	AllDemos  Demos             // All available demos
	Mode      DemoSelectionMode // How to select
	Selector  string            // For SelectByName: demo name/pattern to match
	ConfigDir dt.DirPath        // For SelectByName: where demos are located
	Logger    Logger            // Optional logger
}

// SelectDemos selects demos based on the specified mode
// Returns error if selection fails (e.g., ambiguous match, not found)
func SelectDemos(args *SelectDemosArgs) (selected Demos, err error) {
	var matchResult *MatchDemosResult

	if args == nil {
		err = fmt.Errorf("minion: SelectDemosArgs is nil")
		goto end
	}

	switch args.Mode {
	case SelectByName:
		if args.Selector == "" {
			err = fmt.Errorf("minion: SelectByName requires non-empty Selector")
			goto end
		}

		matchResult, err = MatchDemos(&MatchDemosArgs{
			Selector:  args.Selector,
			ConfigDir: args.ConfigDir,
			Logger:    args.Logger,
		})
		if err != nil {
			err = fmt.Errorf("finding demo: %w", err)
			goto end
		}

		if matchResult.MatchType == NoMatch {
			err = fmt.Errorf("no demo found matching: %q", args.Selector)
			goto end
		}

		if matchResult.MatchType == AmbiguousMatch {
			err = fmt.Errorf("ambiguous demo name %q, matches:\n%s",
				args.Selector, FormatDemoMatches(matchResult.Matches))
			goto end
		}

		selected = matchResult.Matches

	case SelectWithErrors:
		for _, demo := range args.AllDemos {
			if !demo.Valid() {
				selected = append(selected, demo)
			}
		}

	default: // SelectAll
		selected = args.AllDemos
	}

end:
	return selected, err
}

// DemoFormatter handles output formatting for demos
type DemoFormatter struct {
	Writer cliutil.Writer
}

// NewDemoFormatter creates a new DemoFormatter with injected dependencies
func NewDemoFormatter(writer cliutil.Writer) *DemoFormatter {
	return &DemoFormatter{
		Writer: writer,
	}
}

// FormatMode defines the output format
type FormatMode string

const (
	TextFormat FormatMode = "text"
	JSONFormat FormatMode = "json"
)

// FormatDemos outputs selected demos in the specified format
func (f *DemoFormatter) FormatDemos(demos Demos, format FormatMode) (err error) {
	switch format {
	case JSONFormat:
		f.Writer.Printf("%s\n", demos.JSON())
	case TextFormat:
		fallthrough
	default:
		for _, demo := range demos {
			f.formatDemoText(demo)
			f.Writer.Printf("\n")
		}
	}

	return nil
}

// formatDemoText outputs structured validation details for a single demo
func (f *DemoFormatter) formatDemoText(demo *Demo) {
	var validationErrs []string

	f.Writer.Printf("Demo: %s\n", demo.FullName())

	if demo.Valid() {
		f.Writer.Printf("Status: ✓ Valid\n")
	} else {
		f.Writer.Printf("Status: ✗ Invalid\n")
		validationErrs = demo.ValidationErrors()
		if len(validationErrs) > 0 {
			f.Writer.Printf("Issues:\n")
			for _, errMsg := range validationErrs {
				f.Writer.Printf("  - %s\n", errMsg)
			}
		}
	}

	f.Writer.Printf("Install Path: %s\n", demo.InstallPath)
	f.Writer.Printf("Installed: %s\n", demo.Installed.Format("2006-01-02 15:04"))

	if demo.Description != "" {
		f.Writer.Printf("Description: %s\n", demo.Description)
	}
}

// InspectDemosArgs configures the demo inspection
type InspectDemosArgs struct {
	ConfigDir dt.DirPath        // Where demos are stored
	Mode      DemoSelectionMode // How to select demos
	Selector  string            // For SelectByName mode
	Format    FormatMode        // Output format (json or text)
	Writer    cliutil.Writer    // Output writer
	Logger    Logger            // Optional logger
}

// InspectDemos finds, selects, and formats demos for inspection
// Handles all validation, filtering, and formatting in one orchestrated call
func InspectDemos(args *InspectDemosArgs) (err error) {
	var demos Demos
	var selectedDemos Demos
	var formatter *DemoFormatter

	if args == nil {
		err = fmt.Errorf("minion: InspectDemosArgs is nil")
		goto end
	}

	// Find all demos
	demos, err = FindDemos(&FindDemosArgs{
		ConfigDir: args.ConfigDir,
		Logger:    args.Logger,
		CalcSize:  false,
	})
	if err != nil {
		goto end
	}

	// Handle empty demos list
	if len(demos) == 0 {
		args.Writer.Printf("No demos installed\n")
		goto end
	}

	// Select demos based on mode
	selectedDemos, err = SelectDemos(&SelectDemosArgs{
		AllDemos:  demos,
		Mode:      args.Mode,
		Selector:  args.Selector,
		ConfigDir: args.ConfigDir,
		Logger:    args.Logger,
	})
	if err != nil {
		goto end
	}

	// Handle empty selection
	if len(selectedDemos) == 0 && args.Mode == SelectWithErrors {
		args.Writer.Printf("All demos are valid\n")
		goto end
	}

	// Format and output results
	formatter = NewDemoFormatter(args.Writer)
	err = formatter.FormatDemos(selectedDemos, args.Format)

end:
	return err
}
