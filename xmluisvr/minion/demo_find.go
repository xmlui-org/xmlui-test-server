package minion

import (
	"fmt"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// MatchType indicates how the selector matched the demos
type MatchType string

const (
	ExactMatch     MatchType = "exact"
	ContainsMatch  MatchType = "contains"
	WildcardMatch  MatchType = "wildcard"
	AmbiguousMatch MatchType = "ambiguous"
	NoMatch        MatchType = "no_match"
)

// Match represents one segment of a wildcard pattern match
type Match struct {
	Selector string // Part of pattern (e.g., "foo", "*", "???", "baz")
	Value    string // Matched value (e.g., "foo", "_bar_", "b", "a", "r")
}

// MatchDemosArgs specifies parameters for matching demos
type MatchDemosArgs struct {
	Selector  string     // User input: exact name, substring, or pattern
	ConfigDir dt.DirPath // Where demos are stored
	Logger    Logger     // Optional logger
}

// MatchDemosResult contains the results of a demo search
type MatchDemosResult struct {
	Matches      Demos     // Matching demos
	MatchType    MatchType // How the demos were matched
	MatchDetails [][]Match // For each demo, breakdown of how pattern matched
}

// MatchDemos finds demos matching the given selector using exact, contains, and wildcard strategies
func MatchDemos(args *MatchDemosArgs) (result *MatchDemosResult, err error) {
	var demos Demos

	if args == nil || args.Selector == "" {
		result = &MatchDemosResult{
			Matches:   Demos{},
			MatchType: NoMatch,
		}
		return result, nil
	}

	// Get all installed demos
	demos, err = FindDemos(&FindDemosArgs{
		ConfigDir: args.ConfigDir,
		Logger:    args.Logger,
	})
	if err != nil {
		return nil, err
	}

	result = &MatchDemosResult{
		MatchDetails: make([][]Match, 0),
	}

	// Try exact match first
	for _, demo := range demos {
		if demo.FullName() == args.Selector {
			result.Matches = append(result.Matches, demo)
			result.MatchType = ExactMatch
			result.MatchDetails = append(result.MatchDetails, []Match{{args.Selector, demo.FullName()}})
			return result, nil
		}
	}

	// Try contains match
	for _, demo := range demos {
		if strings.Contains(demo.FullName(), args.Selector) {
			result.Matches = append(result.Matches, demo)
			result.MatchType = ContainsMatch
			result.MatchDetails = append(result.MatchDetails, []Match{{args.Selector, args.Selector}})
		}
	}

	if len(result.Matches) > 0 {
		if len(result.Matches) > 1 {
			result.MatchType = AmbiguousMatch
		}
		return result, nil
	}

	// Try wildcard match
	for _, demo := range demos {
		matches, matched := matchWildcard(args.Selector, demo.FullName())
		if matched {
			result.Matches = append(result.Matches, demo)
			result.MatchDetails = append(result.MatchDetails, matches)
		}
	}

	if len(result.Matches) > 0 {
		result.MatchType = WildcardMatch
		if len(result.Matches) > 1 {
			result.MatchType = AmbiguousMatch
		}
		return result, nil
	}

	result.MatchType = NoMatch
	return result, nil
}

// matchWildcard matches a wildcard pattern against a string
// Supports * (matches any number of characters) and ? (matches single character)
// Returns the breakdown of matches and whether it matched
func matchWildcard(pattern string, value string) (matches []Match, matched bool) {
	matches = make([]Match, 0)

	patternIdx := 0
	valueIdx := 0

	for patternIdx < len(pattern) || valueIdx < len(value) {
		if patternIdx >= len(pattern) {
			// Pattern exhausted but value remains
			return matches, false
		}

		patternChar := rune(pattern[patternIdx])

		switch patternChar {
		case '*':
			// * matches any sequence of characters (including empty)
			// Find the next non-wildcard character in pattern
			patternIdx++
			if patternIdx >= len(pattern) {
				// * is at end, matches rest of string
				if valueIdx < len(value) {
					matches = append(matches, Match{"*", value[valueIdx:]})
				} else {
					matches = append(matches, Match{"*", ""})
				}
				matched = true
				return matches, true
			}

			// Find next literal char in pattern
			nextLiteralChar := rune(pattern[patternIdx])
			startVal := valueIdx
			found := false

			for valueIdx < len(value) {
				if rune(value[valueIdx]) == nextLiteralChar {
					found = true
					break
				}
				valueIdx++
			}

			if !found {
				return matches, false
			}

			if valueIdx > startVal {
				matches = append(matches, Match{"*", value[startVal:valueIdx]})
			}

		case '?':
			// ? matches exactly one character
			if valueIdx >= len(value) {
				return matches, false
			}
			matches = append(matches, Match{"?", string(rune(value[valueIdx]))})
			patternIdx++
			valueIdx++

		default:
			// Literal character must match
			if valueIdx >= len(value) || rune(value[valueIdx]) != patternChar {
				return matches, false
			}

			// Collect consecutive literal characters
			startPat := patternIdx
			startVal := valueIdx
			for patternIdx < len(pattern) && valueIdx < len(value) &&
				rune(pattern[patternIdx]) != '*' && rune(pattern[patternIdx]) != '?' &&
				rune(value[valueIdx]) == rune(pattern[patternIdx]) {
				patternIdx++
				valueIdx++
			}

			matches = append(matches, Match{pattern[startPat:patternIdx], value[startVal:valueIdx]})
		}
	}

	return matches, true
}

// FormatDemoMatches formats a list of demos with numbering for user selection
func FormatDemoMatches(demos Demos) string {
	lines := make([]string, len(demos))
	for i, demo := range demos {
		lines[i] = fmt.Sprintf("  %d. %s", i+1, demo.FullName())
	}
	return strings.Join(lines, "\n")
}
