package minion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeschinkel/go-dt"
)

func TestDeleteDemo_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	demoPath := dt.DirPath(filepath.Join(tmpDir, "demo"))
	os.MkdirAll(string(demoPath), 0755)

	demo := &Demo{
		InstallPath: demoPath,
	}

	err := DeleteDemo(&DeleteDemoArgs{
		Demo:   demo,
		DryRun: true,
		Writer: nil,
	})

	if err != nil {
		t.Fatalf("DeleteDemo() error = %v", err)
	}

	// Directory should still exist after dry-run
	if _, err := os.Stat(string(demoPath)); os.IsNotExist(err) {
		t.Error("Demo directory was deleted during dry-run")
	}
}

func TestDeleteDemo_ActualDelete(t *testing.T) {
	tmpDir := t.TempDir()
	demoPath := dt.DirPath(filepath.Join(tmpDir, "demo"))
	os.MkdirAll(string(demoPath), 0755)

	// Create a file inside to ensure recursive delete works
	testFile := filepath.Join(string(demoPath), "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	demo := &Demo{
		InstallPath: demoPath,
	}

	err := DeleteDemo(&DeleteDemoArgs{
		Demo:   demo,
		DryRun: false,
		Writer: nil,
	})

	if err != nil {
		t.Fatalf("DeleteDemo() error = %v", err)
	}

	// Directory should not exist after deletion
	if _, err := os.Stat(string(demoPath)); !os.IsNotExist(err) {
		t.Error("Demo directory was not deleted")
	}
}

func TestDeleteDemo_NonExistent(t *testing.T) {
	demoPath := dt.DirPath("/nonexistent/path/to/demo")

	demo := &Demo{
		InstallPath: demoPath,
	}

	err := DeleteDemo(&DeleteDemoArgs{
		Demo:   demo,
		DryRun: false,
		Writer: nil,
	})

	// Should not error when deleting non-existent directory
	if err != nil {
		t.Fatalf("DeleteDemo() error = %v, want nil", err)
	}
}

func TestDeleteDemo_NilDemo(t *testing.T) {
	err := DeleteDemo(&DeleteDemoArgs{
		Demo:   nil,
		DryRun: false,
		Writer: nil,
	})

	if err != nil {
		t.Fatalf("DeleteDemo() error = %v, want nil", err)
	}
}

func TestDeleteDemo_NilArgs(t *testing.T) {
	err := DeleteDemo(nil)

	if err != nil {
		t.Fatalf("DeleteDemo() error = %v, want nil", err)
	}
}
