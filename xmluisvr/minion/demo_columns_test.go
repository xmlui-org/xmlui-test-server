package minion

import (
	"strings"
	"testing"
	"time"
)

func TestResolveColumn(t *testing.T) {
	var tests = []struct {
		name    string
		input   string
		want    DemoColumn
		wantErr bool
	}{
		{
			name:    "valid column - domain",
			input:   "domain",
			want:    DemoColumnDomain,
			wantErr: false,
		},
		{
			name:    "valid column - typed_ref",
			input:   "typed_ref",
			want:    DemoColumnTypedRef,
			wantErr: false,
		},
		{
			name:    "alias - branch",
			input:   "branch",
			want:    DemoColumnRef,
			wantErr: false,
		},
		{
			name:    "alias - url",
			input:   "url",
			want:    DemoColumnSourceURL,
			wantErr: false,
		},
		{
			name:    "alias - dir",
			input:   "dir",
			want:    DemoColumnInstall,
			wantErr: false,
		},
		{
			name:    "case insensitive",
			input:   "DOMAIN",
			want:    DemoColumnDomain,
			wantErr: false,
		},
		{
			name:    "with whitespace",
			input:   "  domain  ",
			want:    DemoColumnDomain,
			wantErr: false,
		},
		{
			name:    "invalid column",
			input:   "nonexistent",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveColumn(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveColumn(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ResolveColumn(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateColumns(t *testing.T) {
	var tests = []struct {
		name    string
		input   []string
		want    []DemoColumn
		wantErr bool
	}{
		{
			name:    "empty list",
			input:   []string{},
			want:    []DemoColumn{},
			wantErr: false,
		},
		{
			name:    "single valid column",
			input:   []string{"domain"},
			want:    []DemoColumn{DemoColumnDomain},
			wantErr: false,
		},
		{
			name:    "multiple valid columns",
			input:   []string{"domain", "path", "installed"},
			want:    []DemoColumn{DemoColumnDomain, DemoColumnPath, DemoColumnInstalled},
			wantErr: false,
		},
		{
			name:    "with aliases",
			input:   []string{"domain", "branch"},
			want:    []DemoColumn{DemoColumnDomain, DemoColumnRef},
			wantErr: false,
		},
		{
			name:    "single invalid column",
			input:   []string{"invalid"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid",
			input:   []string{"domain", "invalid", "path"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "with whitespace",
			input:   []string{"  domain  ", "path"},
			want:    []DemoColumn{DemoColumnDomain, DemoColumnPath},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateColumns(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateColumns(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if len(got) != len(tt.want) {
				t.Errorf("ValidateColumns(%v) returned %d columns, want %d", tt.input, len(got), len(tt.want))
			}
			for i, col := range got {
				if i >= len(tt.want) {
					break
				}
				if col != tt.want[i] {
					t.Errorf("ValidateColumns(%v)[%d] = %q, want %q", tt.input, i, col, tt.want[i])
				}
			}
		})
	}
}

func TestGetAllColumns(t *testing.T) {
	got := GetAllColumns()

	if len(got) == 0 {
		t.Fatal("GetAllColumns() returned empty list, expected at least one column")
	}

	// Verify all returned columns have metadata
	for _, col := range got {
		meta := GetColumnMeta(col)
		if meta == nil {
			t.Errorf("GetAllColumns() returned column %q with no metadata", col)
		}
	}
}

func TestGetColumnMeta(t *testing.T) {
	var tests = []struct {
		name    string
		col     DemoColumn
		wantNil bool
	}{
		{
			name:    "domain column",
			col:     DemoColumnDomain,
			wantNil: false,
		},
		{
			name:    "typed_ref column",
			col:     DemoColumnTypedRef,
			wantNil: false,
		},
		{
			name:    "invalid column",
			col:     "nonexistent",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetColumnMeta(tt.col)
			if (got == nil) != tt.wantNil {
				t.Errorf("GetColumnMeta(%q) returned nil=%v, wantNil=%v", tt.col, got == nil, tt.wantNil)
			}
			if !tt.wantNil && got != nil {
				if got.ID != tt.col {
					t.Errorf("GetColumnMeta(%q) returned meta with ID %q", tt.col, got.ID)
				}
			}
		})
	}
}

func TestFormatInvalidColumnError(t *testing.T) {
	errMsg := "invalid column at position 0: unknown column: foo"
	got := FormatInvalidColumnError(errMsg)

	if !strings.Contains(got, errMsg) {
		t.Errorf("FormatInvalidColumnError() should contain original error message")
	}

	if !strings.Contains(got, "Available columns:") {
		t.Errorf("FormatInvalidColumnError() should contain 'Available columns:'")
	}

	if !strings.Contains(got, "domain") {
		t.Errorf("FormatInvalidColumnError() should list domain column")
	}
}

func TestFormatAge(t *testing.T) {
	var now time.Time
	var tests = []struct {
		name string
		age  time.Duration
		want string
	}{
		{
			name: "now",
			age:  0,
			want: "now",
		},
		{
			name: "minutes",
			age:  5 * time.Minute,
			want: "5m",
		},
		{
			name: "hours",
			age:  3 * time.Hour,
			want: "3h",
		},
		{
			name: "days",
			age:  5 * 24 * time.Hour,
			want: "5d",
		},
		{
			name: "months",
			age:  60 * 24 * time.Hour,
			want: "2mo",
		},
	}

	now = time.Now()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			past := now.Add(-tt.age)
			got := formatAge(past, now)

			if got != tt.want {
				t.Errorf("formatAge(%v ago) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	var tests = []struct {
		name string
		size int64
		want string
	}{
		{
			name: "bytes",
			size: 512,
			want: "512 B",
		},
		{
			name: "kilobytes",
			size: 2 * 1024,
			want: "2.0 KB",
		},
		{
			name: "megabytes",
			size: 5 * 1024 * 1024,
			want: "5.0 MB",
		},
		{
			name: "gigabytes",
			size: 1024 * 1024 * 1024,
			want: "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.size)

			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.size, got, tt.want)
			}
		})
	}
}
