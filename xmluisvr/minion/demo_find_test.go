package minion

import (
	"testing"
)

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		// Exact matches
		{"exact match", "foo", "foo", true},
		{"exact no match", "foo", "bar", false},

		// Star wildcard tests
		{"star at end", "foo*", "foobar", true},
		{"star at end no match", "foo*", "barfoo", false},
		{"star matches empty", "foo*", "foo", true},
		{"star in middle", "foo*baz", "foo_bar_baz", true},
		{"star in middle no match", "foo*baz", "foo_bar_qux", false},
		{"star at start", "*baz", "foo_bar_baz", true},
		{"star at start no match", "*baz", "foo_bar_qux", false},
		{"multiple stars", "foo*bar*baz", "foo_x_bar_y_baz", true},

		// Question mark wildcard tests
		{"single question mark", "fo?", "foo", true},
		{"single question mark no match", "fo?", "foooo", false},
		{"multiple question marks", "f??", "foo", true},
		{"multiple question marks no match", "f??", "foo", false},

		// Combined tests
		{"star and question", "f?o*baz", "foobar_baz", true},
		{"question and star", "f?o*baz", "foxbarbaz", true},
		{"question and star no match", "f?o*baz", "fxobarbaz", false},

		// Edge cases
		{"empty pattern empty value", "", "", true},
		{"pattern longer than value", "abcdef", "abc", false},
		{"value longer than pattern no wildcard", "abc", "abcdef", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := matchWildcard(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchWildcardDetails(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		value       string
		wantMatched bool
		wantLen     int // Expected number of Match elements
	}{
		{"foo*baz matching foo_bar_baz", "foo*baz", "foo_bar_baz", true, 3},
		{"foo_???_baz matching foo_bar_baz", "foo_???_baz", "foo_bar_baz", true, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, got := matchWildcard(tt.pattern, tt.value)
			if got != tt.wantMatched {
				t.Errorf("matchWildcard(%q, %q) matched = %v, want %v", tt.pattern, tt.value, got, tt.wantMatched)
			}
			if got && len(matches) != tt.wantLen {
				t.Errorf("matchWildcard(%q, %q) returned %d matches, want %d", tt.pattern, tt.value, len(matches), tt.wantLen)
			}
		})
	}
}

// TestMatchDemos_ExactMatch tests exact matching - requires real demo list
// Skipped in unit test env as it needs FindDemos() to work

func TestFormatDemoMatches(t *testing.T) {
	demos := Demos{
		&Demo{
			Domain:  "github.com",
			Org:     "xmlui-org",
			Repo:    "xmlui-todo",
			Ref:     "main",
			RefType: BranchRefType,
		},
		&Demo{
			Domain:  "github.com",
			Org:     "xmlui-org",
			Repo:    "xmlui-hello",
			Ref:     "v1.0.0",
			RefType: TagRefType,
		},
	}

	output := FormatDemoMatches(demos)

	// Check that output contains numbered lines
	if output == "" {
		t.Error("FormatDemoMatches() returned empty string")
	}

	// Should contain numbers
	if !contains(output, "1.") || !contains(output, "2.") {
		t.Error("FormatDemoMatches() output doesn't contain expected numbering")
	}

	// Should contain full names
	if !contains(output, "github.com/xmlui-org/xmlui-todo#main") {
		t.Error("FormatDemoMatches() output doesn't contain first demo name")
	}

	if !contains(output, "github.com/xmlui-org/xmlui-hello#v1.0.0") {
		t.Error("FormatDemoMatches() output doesn't contain second demo name")
	}
}

func contains(s, substring string) bool {
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
