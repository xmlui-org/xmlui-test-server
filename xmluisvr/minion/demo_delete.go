package minion

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"golang.org/x/term"
)

// DeleteDemoArgs specifies parameters for deleting a demo
type DeleteDemoArgs struct {
	Demo   *Demo          // The demo to delete
	DryRun bool           // If true, show what would be deleted without actually deleting
	Writer cliutil.Writer // For output messages
}

// DeleteDemo deletes the specified demo directory
func DeleteDemo(args *DeleteDemoArgs) (err error) {
	if args == nil || args.Demo == nil {
		return nil
	}

	// Validate the demo's install path exists
	exists, err := args.Demo.InstallPath.Exists()
	if err != nil {
		return err
	}

	if !exists {
		// Directory already doesn't exist, nothing to do
		return nil
	}

	if args.DryRun {
		if args.Writer != nil {
			args.Writer.Printf("Would delete: %s\n", args.Demo.InstallPath)
		}
		return nil
	}

	// Delete the directory and all contents
	err = os.RemoveAll(string(args.Demo.InstallPath))
	if err != nil {
		return err
	}

	return nil
}

// DeleteDemosArgs specifies parameters for potentially multiple demo deletion
type DeleteDemosArgs struct {
	Selector  string         // User input: demo selector
	ConfigDir string         // Where demos are stored
	Force     bool           // Skip all prompts (requires exact match)
	DryRun    bool           // Show what would be deleted without actually deleting
	Writer    cliutil.Writer // For output/prompts
	Logger    Logger         // Optional logger
}

// DeleteDemos finds a demo, prompts for confirmation, and deletes it
// Returns the deleted demo for informational purposes
func DeleteDemos(args *DeleteDemosArgs) (deleted *Demo, err error) {
	var matchResult *MatchDemosResult
	var selectedDemo *Demo

	// 1. Find matching demos
	matchResult, err = MatchDemos(&MatchDemosArgs{
		Selector:  args.Selector,
		ConfigDir: dt.DirPath(args.ConfigDir),
		Logger:    args.Logger,
	})
	if err != nil {
		return nil, err
	}

	// 2. Handle match results
	selectedDemo, err = selectDemoForDeletion(matchResult, args.Selector, args.Force, args.Writer)
	if err != nil {
		return nil, err
	}

	// 3. Confirm deletion (unless --force)
	err = confirmDeletion(selectedDemo, args.Force, args.Writer)
	if err != nil {
		return nil, err
	}

	// 4. Delete demo
	err = DeleteDemo(&DeleteDemoArgs{
		Demo:   selectedDemo,
		DryRun: args.DryRun,
		Writer: args.Writer,
	})
	if err != nil {
		return nil, err
	}

	return selectedDemo, nil
}

// selectDemoForDeletion handles demo matching and selection
func selectDemoForDeletion(result *MatchDemosResult, selector string, force bool, writer cliutil.Writer) (selected *Demo, err error) {
	switch result.MatchType {
	case NoMatch:
		if writer != nil {
			writer.Errorf("No demo found matching: %s\n", selector)
		}
		return nil, nil // Return nil to signal not found

	case ExactMatch:
		return result.Matches[0], nil

	case AmbiguousMatch:
		if force {
			// Force with ambiguous = error with helpful message
			if writer != nil {
				writer.Errorf("Ambiguous selector '%s' matches multiple demos:\n", selector)
				writer.Printf("%s\n", FormatDemoMatches(result.Matches))
				writer.Printf("\nWith --force, you must use the full demo identifier:\n")
				for _, d := range result.Matches {
					writer.Printf("  xmlui demo delete --force %s\n", d.FullName())
				}
			}
			return nil, nil // Return nil to signal ambiguous with force
		}
		// Interactive: prompt for selection
		return promptUserSelection(result.Matches, writer)

	case ContainsMatch, WildcardMatch:
		if len(result.Matches) == 1 {
			return result.Matches[0], nil
		}
		// Multiple matches
		if force {
			if writer != nil {
				writer.Errorf("Ambiguous selector '%s' matches multiple demos:\n", selector)
				writer.Printf("%s\n", FormatDemoMatches(result.Matches))
				writer.Printf("\nWith --force, you must use the full demo identifier:\n")
				for _, d := range result.Matches {
					writer.Printf("  xmlui demo delete --force %s\n", d.FullName())
				}
			}
			return nil, nil
		}
		return promptUserSelection(result.Matches, writer)
	}

	return nil, nil
}

// confirmDeletion prompts the user to confirm deletion
func confirmDeletion(demo *Demo, force bool, writer cliutil.Writer) (err error) {
	if force {
		return nil // Skip confirmation with --force
	}

	if writer != nil {
		writer.Printf("Delete %s? [y/N]: ", demo.FullName())
	}

	confirmed, err := promptYesNo()
	if err != nil {
		return err
	}

	if !confirmed {
		if writer != nil {
			writer.Printf("Deletion cancelled\n")
		}
		return nil // Return nil to signal user cancelled
	}

	return nil
}

// promptUserSelection prompts user to select from multiple demos
func promptUserSelection(demos Demos, writer cliutil.Writer) (selected *Demo, err error) {
	var input string
	var selection int

	// Check if in terminal for interactive mode
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		if writer != nil {
			writer.Errorf("Interactive selection required but not in terminal\n")
		}
		return nil, nil
	}

	if writer != nil {
		writer.Printf("Found %d demos matching selector:\n", len(demos))
		writer.Printf("%s\n", FormatDemoMatches(demos))
		writer.Printf("Select demo to delete [1-%d, or Enter to cancel]: ", len(demos))
	}

	input, err = readLine()
	if err != nil {
		if err == io.EOF {
			if writer != nil {
				writer.Printf("\nDeletion cancelled\n")
			}
			return nil, nil
		}
		return nil, err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		if writer != nil {
			writer.Printf("Deletion cancelled\n")
		}
		return nil, nil
	}

	selection, err = strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(demos) {
		if writer != nil {
			writer.Errorf("Invalid selection: %s\n", input)
		}
		return nil, nil
	}

	return demos[selection-1], nil
}

// promptYesNo prompts for yes/no response
func promptYesNo() (confirmed bool, err error) {
	input, err := readLine()
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes", nil
}

// readLine reads a line from stdin
func readLine() (line string, err error) {
	reader := bufio.NewReader(os.Stdin)
	line, err = reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")

	return line, nil
}
