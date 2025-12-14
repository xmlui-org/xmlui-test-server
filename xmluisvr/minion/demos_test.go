package minion

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-fsfix"
)

// TestDemoFullName tests Demo.FullName() formatting for GitHub and URL sources
func TestDemoFullName(t *testing.T) {
	tests := []struct {
		name     string
		demo     *Demo
		wantName string
	}{
		{
			name: "GitHub demo with branch",
			demo: &Demo{
				Domain:  "github.com",
				Org:     "xmlui-org",
				Repo:    "xmlui-hello",
				Ref:     "main",
				RefType: BranchRefType,
			},
			wantName: "github.com/xmlui-org/xmlui-hello#main",
		},
		{
			name: "GitHub demo with tag",
			demo: &Demo{
				Domain:  "github.com",
				Org:     "xmlui-org",
				Repo:    "xmlui-hello",
				Ref:     "v1.0.0",
				RefType: TagRefType,
			},
			wantName: "github.com/xmlui-org/xmlui-hello#v1.0.0",
		},
		{
			name: "GitHub demo with commit hash",
			demo: &Demo{
				Domain:  "github.com",
				Org:     "other-org",
				Repo:    "other-repo",
				Ref:     "abc123def456",
				RefType: HashRefType,
			},
			wantName: "github.com/other-org/other-repo#abc123def456",
		},
		{
			name: "URL-based demo",
			demo: &Demo{
				Domain:    "example.com",
				SourceURL: "https://example.com/demo.zip",
				RefType:   URLRefType,
			},
			wantName: "example.com/https://example.com/demo.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.demo.FullName()
			if got != tt.wantName {
				t.Errorf("FullName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

// TestDemoJSON tests Demo.JSON() serialization
func TestDemoJSON(t *testing.T) {
	tests := []struct {
		name         string
		demo         *Demo
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "Nil demo",
			demo:         nil,
			wantContains: []string{"null"},
		},
		{
			name: "Valid demo JSON",
			demo: &Demo{
				Domain:  "github.com",
				Org:     "xmlui-org",
				Repo:    "xmlui-hello",
				Ref:     "main",
				RefType: BranchRefType,
			},
			wantContains: []string{"github.com", "xmlui-org", "xmlui-hello", "main", "branch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.demo.JSON()
			for _, substr := range tt.wantContains {
				if !strings.Contains(got, substr) {
					t.Errorf("JSON() = %q, should contain %q", got, substr)
				}
			}
		})
	}
}

// TestDemosFullNames tests Demos.FullNames() slice extraction
func TestDemosFullNames(t *testing.T) {
	tests := []struct {
		name     string
		demos    Demos
		wantLen  int
		wantName string
	}{
		{
			name:    "Empty demos",
			demos:   Demos{},
			wantLen: 0,
		},
		{
			name: "Single demo",
			demos: Demos{
				&Demo{
					Domain:  "github.com",
					Org:     "org1",
					Repo:    "repo1",
					Ref:     "main",
					RefType: BranchRefType,
				},
			},
			wantLen:  1,
			wantName: "github.com/org1/repo1#main",
		},
		{
			name: "Multiple demos",
			demos: Demos{
				&Demo{
					Domain:  "github.com",
					Org:     "org1",
					Repo:    "repo1",
					Ref:     "main",
					RefType: BranchRefType,
				},
				&Demo{
					Domain:  "github.com",
					Org:     "org2",
					Repo:    "repo2",
					Ref:     "develop",
					RefType: BranchRefType,
				},
			},
			wantLen: 2,
		},
		{
			name:    "Demos with nil entries",
			demos:   Demos{nil, &Demo{Domain: "github.com", Org: "org1", Repo: "repo1", Ref: "main", RefType: BranchRefType}, nil},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.demos.FullNames()
			if len(got) != tt.wantLen {
				t.Fatalf("FullNames() len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantName != "" && len(got) > 0 && got[0] != tt.wantName {
				t.Errorf("FullNames()[0] = %q, want %q", got[0], tt.wantName)
			}
		})
	}
}

// TestDemosJSON tests Demos.JSON() collection serialization
func TestDemosJSON(t *testing.T) {
	tests := []struct {
		name         string
		demos        Demos
		wantContains []string
	}{
		{
			name:         "Empty demos",
			demos:        Demos{},
			wantContains: []string{"[]"},
		},
		{
			name: "Single demo with validation",
			demos: Demos{
				&Demo{
					Domain:  "github.com",
					Org:     "xmlui-org",
					Repo:    "xmlui-hello",
					Ref:     "main",
					RefType: BranchRefType,
				},
			},
			wantContains: []string{"github.com", "valid"},
		},
		{
			name: "Multiple demos",
			demos: Demos{
				&Demo{
					Domain:  "github.com",
					Org:     "org1",
					Repo:    "repo1",
					Ref:     "main",
					RefType: BranchRefType,
				},
				&Demo{
					Domain:  "github.com",
					Org:     "org2",
					Repo:    "repo2",
					Ref:     "develop",
					RefType: BranchRefType,
				},
			},
			wantContains: []string{"[", "]", "org1", "org2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.demos.JSON()
			for _, substr := range tt.wantContains {
				if !strings.Contains(got, substr) {
					t.Errorf("JSON() = %q, should contain %q", got, substr)
				}
			}
		})
	}
}

// TestFindDemos tests demo discovery from filesystem
func TestFindDemos(t *testing.T) {
	tests := []struct {
		name         string
		setupFixture func(t *testing.T) (tf *fsfix.RootFixture, configDir dt.DirPath)
		sortBy       DemoSort
		sortDesc     bool
		wantCount    int
		wantFirstOrg dt.PathSegment
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name: "Empty demos directory",
			setupFixture: func(t *testing.T) (*fsfix.RootFixture, dt.DirPath) {
				tf := fsfix.NewRootFixture("empty-demos")
				defer tf.Cleanup()

				configDir := tf.AddDirFixture(t, ".config", nil)
				tf.Create(t)

				return tf, dt.DirPath(configDir.Filepath)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "Single valid GitHub demo",
			setupFixture: func(t *testing.T) (*fsfix.RootFixture, dt.DirPath) {
				tf := fsfix.NewRootFixture("single-github-demo")
				defer tf.Cleanup()

				configDir := tf.AddDirFixture(t, ".config", nil)
				demosDir := configDir.AddDirFixture(t, "demos", nil)
				githubDir := demosDir.AddDirFixture(t, "github.com", nil)
				orgDir := githubDir.AddDirFixture(t, "xmlui-org", nil)
				repoDir := orgDir.AddDirFixture(t, "xmlui-hello", nil)
				archiveDir := repoDir.AddDirFixture(t, "archive", nil)
				mainDir := archiveDir.AddDirFixture(t, "main", nil)

				mainDir.AddFileFixture(t, "index.html", &fsfix.FileFixtureArgs{
					Content: `<html><script src="xmlui.bundle.js"></script></html>`,
				})
				mainDir.AddFileFixture(t, "Main.xmlui", &fsfix.FileFixtureArgs{
					Content: `<App title="Hello"></App>`,
				})
				mainDir.AddFileFixture(t, "README.md", &fsfix.FileFixtureArgs{
					Content: "# Hello Demo\nA simple demo",
				})

				tf.Create(t)
				return tf, dt.DirPath(configDir.Filepath)
			},
			wantCount:    1,
			wantFirstOrg: "xmlui-org",
			wantErr:      false,
		},
		{
			name: "Multiple demos with sorting by name",
			setupFixture: func(t *testing.T) (*fsfix.RootFixture, dt.DirPath) {
				tf := fsfix.NewRootFixture("multi-demos-sort-name")
				defer tf.Cleanup()

				configDir := tf.AddDirFixture(t, ".config", nil)
				demosDir := configDir.AddDirFixture(t, "demos", nil)
				githubDir := demosDir.AddDirFixture(t, "github.com", nil)

				// Create org1/repo-a
				org1Dir := githubDir.AddDirFixture(t, "org1", nil)
				repoADir := org1Dir.AddDirFixture(t, "repo-a", nil)
				archiveADir := repoADir.AddDirFixture(t, "archive", nil)
				mainADir := archiveADir.AddDirFixture(t, "main", nil)
				mainADir.AddFileFixture(t, "index.html", &fsfix.FileFixtureArgs{
					Content: `<html></html>`,
				})
				mainADir.AddFileFixture(t, "Main.xmlui", &fsfix.FileFixtureArgs{
					Content: `<App></App>`,
				})

				// Create org2/repo-b
				org2Dir := githubDir.AddDirFixture(t, "org2", nil)
				repoBDir := org2Dir.AddDirFixture(t, "repo-b", nil)
				archiveBDir := repoBDir.AddDirFixture(t, "archive", nil)
				mainBDir := archiveBDir.AddDirFixture(t, "main", nil)
				mainBDir.AddFileFixture(t, "index.html", &fsfix.FileFixtureArgs{
					Content: `<html></html>`,
				})
				mainBDir.AddFileFixture(t, "Main.xmlui", &fsfix.FileFixtureArgs{
					Content: `<App></App>`,
				})

				tf.Create(t)
				return tf, dt.DirPath(configDir.Filepath)
			},
			sortBy:    NameSort,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "Nonexistent config directory",
			setupFixture: func(t *testing.T) (*fsfix.RootFixture, dt.DirPath) {
				tf := fsfix.NewRootFixture("nonexistent-config")
				tf.Create(t)
				return tf, dt.DirPath("/nonexistent/config/path")
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf, configDir := tt.setupFixture(t)
			defer tf.Cleanup()

			got, err := FindDemos(&FindDemosArgs{
				ConfigDir: configDir,
				SortBy:    tt.sortBy,
				SortDesc:  tt.sortDesc,
			})

			if (err != nil) != tt.wantErr {
				t.Fatalf("FindDemos() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error = %q, want substring %q", err, tt.wantErrMsg)
			}

			if len(got) != tt.wantCount {
				t.Errorf("FindDemos() len = %d, want %d", len(got), tt.wantCount)
			}

			if tt.wantFirstOrg != "" && len(got) > 0 && got[0].Org != tt.wantFirstOrg {
				t.Errorf("FindDemos()[0].Org = %q, want %q", got[0].Org, tt.wantFirstOrg)
			}
		})
	}
}

// TestParseGitHubPath tests path parsing for GitHub repos
func TestParseGitHubPath(t *testing.T) {
	tests := []struct {
		name        string
		path        dt.DirPath
		wantOrg     dt.PathSegment
		wantRepo    dt.PathSegment
		wantRef     dt.Identifier
		wantRefType RefType
	}{
		{
			name:        "Standard main branch",
			path:        "xmlui-org/xmlui-hello/archive/main",
			wantOrg:     "xmlui-org",
			wantRepo:    "xmlui-hello",
			wantRef:     "main",
			wantRefType: BranchRefType,
		},
		{
			name:        "Version tag",
			path:        "myorg/myrepo/archive/v1.0.0",
			wantOrg:     "myorg",
			wantRepo:    "myrepo",
			wantRef:     "v1.0.0",
			wantRefType: BranchRefType, // Default when can't determine
		},
		{
			name:        "Develop branch",
			path:        "org/repo/archive/develop",
			wantOrg:     "org",
			wantRepo:    "repo",
			wantRef:     "develop",
			wantRefType: BranchRefType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, ref := parseGitHubPath(tt.path)

			if org != tt.wantOrg {
				t.Errorf("org = %q, want %q", org, tt.wantOrg)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}

// TestSortDemos tests demo sorting by name, date, and domain
func TestSortDemos(t *testing.T) {
	tests := []struct {
		name   string
		demos  Demos
		sortBy DemoSort
		desc   bool
		want   []string // Expected order of FullNames
	}{
		{
			name: "Sort by name ascending",
			demos: Demos{
				&Demo{Domain: "github.com", Org: "b", Repo: "repo", Ref: "main", RefType: BranchRefType},
				&Demo{Domain: "github.com", Org: "a", Repo: "repo", Ref: "main", RefType: BranchRefType},
				&Demo{Domain: "github.com", Org: "c", Repo: "repo", Ref: "main", RefType: BranchRefType},
			},
			sortBy: NameSort,
			desc:   false,
			want: []string{
				"github.com/a/repo#main",
				"github.com/b/repo#main",
				"github.com/c/repo#main",
			},
		},
		{
			name: "Sort by name descending",
			demos: Demos{
				&Demo{Domain: "github.com", Org: "a", Repo: "repo", Ref: "main", RefType: BranchRefType},
				&Demo{Domain: "github.com", Org: "c", Repo: "repo", Ref: "main", RefType: BranchRefType},
				&Demo{Domain: "github.com", Org: "b", Repo: "repo", Ref: "main", RefType: BranchRefType},
			},
			sortBy: NameSort,
			desc:   true,
			want: []string{
				"github.com/c/repo#main",
				"github.com/b/repo#main",
				"github.com/a/repo#main",
			},
		},
		{
			name: "Sort by domain",
			demos: Demos{
				&Demo{Domain: "example.com", SourceURL: "url1", RefType: URLRefType},
				&Demo{Domain: "github.com", Org: "org", Repo: "repo", Ref: "main", RefType: BranchRefType},
				&Demo{Domain: "custom.com", SourceURL: "url2", RefType: URLRefType},
			},
			sortBy: DomainSort,
			desc:   false,
			want: []string{
				"custom.com/url2",
				"example.com/url1",
				"github.com/org/repo#main",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sort in place
			sortDemos(tt.demos, tt.sortBy, tt.desc)

			got := tt.demos.FullNames()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("After sort, got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractDescription tests README extraction
func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name             string
		setupFixture     func(t *testing.T) dt.DirPath
		wantDescContains string
	}{
		{
			name: "README.md with markdown heading",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("readme-md")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				demoDir.AddFileFixture(t, "README.md", &fsfix.FileFixtureArgs{
					Content: "# My Demo App\nThis is the description",
				})
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantDescContains: "My Demo App",
		},
		{
			name: "README file (no extension)",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("readme-plain")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				demoDir.AddFileFixture(t, "README", &fsfix.FileFixtureArgs{
					Content: "# Plain README\nContent here",
				})
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantDescContains: "Plain README",
		},
		{
			name: "No README file",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("no-readme")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo-name", nil)
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantDescContains: "demo-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			demoDir := tt.setupFixture(t)

			got := extractDescription(demoDir)

			if !strings.Contains(got, tt.wantDescContains) {
				t.Errorf("extractDescription() = %q, should contain %q", got, tt.wantDescContains)
			}
		})
	}
}

// TestDemoValid tests demo validation with fsfix
func TestDemoValid(t *testing.T) {
	tests := []struct {
		name         string
		setupFixture func(t *testing.T) dt.DirPath
		wantValid    bool
	}{
		{
			name: "Valid demo with required files",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("valid-demo")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				demoDir.AddFileFixture(t, "index.html", &fsfix.FileFixtureArgs{
					Content: `<html></html>`,
				})
				demoDir.AddFileFixture(t, "Main.xmlui", &fsfix.FileFixtureArgs{
					Content: `<App></App>`,
				})
				demoDir.AddFileFixture(t, "config.json", &fsfix.FileFixtureArgs{
					Content: `{"version": 1}`,
				})
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantValid: true,
		},
		{
			name: "Invalid demo missing index.html",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("missing-index")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				demoDir.AddFileFixture(t, "Main.xmlui", &fsfix.FileFixtureArgs{
					Content: `<App></App>`,
				})
				demoDir.AddFileFixture(t, "config.json", &fsfix.FileFixtureArgs{
					Content: `{"version": 1}`,
				})
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantValid: false,
		},
		{
			name: "Invalid demo missing Main.xmlui",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("missing-main-xmlui")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				demoDir.AddFileFixture(t, "index.html", &fsfix.FileFixtureArgs{
					Content: `<html></html>`,
				})
				demoDir.AddFileFixture(t, "config.json", &fsfix.FileFixtureArgs{
					Content: `{"version": 1}`,
				})
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantValid: false,
		},
		{
			name: "Empty directory not valid",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("empty-demo")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			demoDir := tt.setupFixture(t)

			demo := &Demo{InstallPath: demoDir}
			got := demo.Valid()

			if got != tt.wantValid {
				t.Errorf("Valid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

// TestDemoValidationErrors tests error collection for invalid demos
func TestDemoValidationErrors(t *testing.T) {
	tests := []struct {
		name                    string
		setupFixture            func(t *testing.T) dt.DirPath
		wantHasValidationErrors bool
	}{
		{
			name: "Valid demo has no errors",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("valid-demo-no-errors")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				demoDir.AddFileFixture(t, "index.html", &fsfix.FileFixtureArgs{
					Content: `<html></html>`,
				})
				demoDir.AddFileFixture(t, "Main.xmlui", &fsfix.FileFixtureArgs{
					Content: `<App></App>`,
				})
				demoDir.AddFileFixture(t, "config.json", &fsfix.FileFixtureArgs{
					Content: `{"version": 1}`,
				})
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantHasValidationErrors: false,
		},
		{
			name: "Invalid demo has errors",
			setupFixture: func(t *testing.T) dt.DirPath {
				tf := fsfix.NewRootFixture("invalid-demo-has-errors")
				defer tf.Cleanup()

				demoDir := tf.AddDirFixture(t, "demo", nil)
				tf.Create(t)
				return dt.DirPath(demoDir.Filepath)
			},
			wantHasValidationErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			demoDir := tt.setupFixture(t)

			demo := &Demo{InstallPath: demoDir}
			got := demo.ValidationErrors()

			hasErrors := len(got) > 0
			if hasErrors != tt.wantHasValidationErrors {
				t.Errorf("ValidationErrors() has errors = %v, want %v (errors: %v)", hasErrors, tt.wantHasValidationErrors, got)
			}
		})
	}
}

// TestFormatSize tests human-readable size formatting
func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "Zero bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "512 bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "1 kilobyte",
			bytes: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "1.5 megabytes",
			bytes: 1536 * 1024,
			want:  "1.5 MB",
		},
		{
			name:  "1 gigabyte",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// TestFormatAge tests time-since-now formatting
func TestFormatAge(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		time time.Time
		now  time.Time
		want string
	}{
		{
			name: "Just now",
			time: now,
			now:  now,
			want: "now",
		},
		{
			name: "5 minutes ago",
			time: now.Add(-5 * time.Minute),
			now:  now,
			want: "5m",
		},
		{
			name: "2 hours ago",
			time: now.Add(-2 * time.Hour),
			now:  now,
			want: "2h",
		},
		{
			name: "7 days ago",
			time: now.Add(-7 * 24 * time.Hour),
			now:  now,
			want: "7d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.time, tt.now)
			if got != tt.want {
				t.Errorf("formatAge() = %q, want %q", got, tt.want)
			}
		})
	}
}
