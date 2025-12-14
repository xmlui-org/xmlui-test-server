package minion

import (
	"fmt"
	"time"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

// DemoListMode defines how to display demos
type DemoListMode string

const (
	ListModeTable  DemoListMode = "table"
	ListModeSimple DemoListMode = "simple"
	ListModeJSON   DemoListMode = "json"
)

// DemoListFormatter handles formatting and output for demo lists
type DemoListFormatter struct {
	Writer cliutil.Writer
}

// NewDemoListFormatter creates a new DemoListFormatter with injected dependencies
func NewDemoListFormatter(writer cliutil.Writer) *DemoListFormatter {
	return &DemoListFormatter{
		Writer: writer,
	}
}

// FormatDemoList outputs the demo list in the specified mode
func (f *DemoListFormatter) FormatDemoList(demos Demos, mode DemoListMode, args *DemoListFormatArgs) (err error) {
	switch mode {
	case ListModeJSON:
		err = f.formatJSON(demos)

	case ListModeSimple:
		err = f.formatSimple(demos)

	default: // ListModeTable
		err = f.formatTable(demos, args)
	}

	return err
}

// DemoListFormatArgs configures table formatting options
type DemoListFormatArgs struct {
	SortBy     DemoSort
	SortDesc   bool
	TimeFormat dt.TimeFormat
	DemosDir   dt.DirPath
	HomeDir    dt.DirPath
	ShowSize   bool
	Columns    []DemoColumn
}

// formatJSON outputs demos as JSON
func (f *DemoListFormatter) formatJSON(demos Demos) (err error) {
	if len(demos) == 0 {
		f.Writer.Printf("[]\n")
	} else {
		f.Writer.Printf("%s\n", demos.JSON())
	}
	return nil
}

// formatSimple outputs demos as a simple name-only list
func (f *DemoListFormatter) formatSimple(demos Demos) (err error) {
	for _, name := range demos.FullNames() {
		f.Writer.Printf("%s\n", name)
	}
	return nil
}

// formatTable outputs demos as a formatted table
func (f *DemoListFormatter) formatTable(demos Demos, args *DemoListFormatArgs) (err error) {
	var timeFormat dt.TimeFormat
	var hasInvalid bool
	var demo *Demo

	if args == nil {
		args = &DemoListFormatArgs{}
	}

	timeFormat = args.TimeFormat
	if timeFormat == "" {
		timeFormat = time.DateOnly
	}

	tw := demos.TableWriter(DemoTableWriterArgs{
		SortBy:     args.SortBy,
		SortDesc:   args.SortDesc,
		TimeFormat: timeFormat,
		DemosDir:   args.DemosDir,
		HomeDir:    args.HomeDir,
		ShowIndex:  true,
		ShowSize:   args.ShowSize,
		Columns:    args.Columns,
	})

	f.Writer.Printf("%s\n", tw.Render())

	// Check if any demos are invalid and show hint
	for _, demo = range demos {
		if demo != nil && !demo.Valid() {
			hasInvalid = true
			break
		}
	}

	if hasInvalid {
		f.Writer.Printf("\nRun 'xmlui demo inspect --with-errors' for details on invalid demos\n")
	}

	return nil
}

// ListDemosArgs configures the demo listing operation
type ListDemosArgs struct {
	ConfigDir   dt.DirPath   // Where demos are stored
	HomeDir     dt.DirPath   // User home directory
	Mode        DemoListMode // Output mode (table, simple, json)
	SortBy      DemoSort     // Sort order
	SortDesc    bool         // Reverse sort
	CalcSize    bool         // Calculate directory sizes
	ShowSize    bool         // Include size in output
	ColumnNames []string     // Comma-separated column names (for table mode)
	Writer      cliutil.Writer
	Logger      Logger
}

// ListDemos finds, optionally validates columns, and formats demos for listing
func ListDemos(args *ListDemosArgs) (err error) {
	var demos Demos
	var columns []DemoColumn
	var formatter *DemoListFormatter

	if args == nil {
		err = fmt.Errorf("minion: ListDemosArgs is nil")
		goto end
	}

	// Validate and parse columns if provided
	if len(args.ColumnNames) > 0 {
		columns, err = ValidateColumns(args.ColumnNames)
		if err != nil {
			args.Writer.Errorf("%s\n", FormatInvalidColumnError(err.Error()))
			err = fmt.Errorf("invalid columns: %w", err)
			goto end
		}
	}

	// Find all demos
	demos, err = FindDemos(&FindDemosArgs{
		ConfigDir: args.ConfigDir,
		Logger:    args.Logger,
		SortBy:    args.SortBy,
		SortDesc:  args.SortDesc,
		CalcSize:  args.CalcSize,
	})
	if err != nil {
		err = fmt.Errorf("collecting demos: %w", err)
		goto end
	}

	// Handle empty demos list
	if len(demos) == 0 {
		if args.Mode == ListModeJSON {
			args.Writer.Printf("[]\n")
		} else {
			args.Writer.Printf("No demos installed\n")
		}
		goto end
	}

	// Format and output results
	formatter = NewDemoListFormatter(args.Writer)
	err = formatter.FormatDemoList(demos, args.Mode, &DemoListFormatArgs{
		SortBy:     args.SortBy,
		SortDesc:   args.SortDesc,
		TimeFormat: time.DateOnly,
		DemosDir:   dt.DirPathJoin(args.ConfigDir, "demos"), // Will be set by caller
		HomeDir:    args.HomeDir,
		ShowSize:   args.ShowSize,
		Columns:    columns,
	})

end:
	return err
}
