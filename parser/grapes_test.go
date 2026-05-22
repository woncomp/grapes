package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGrapesFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "shell.toml", `
[[grape]]
import = "path"

[[grape]]
from = "shared"
import = "prompt.grape"
`)

	grapes, err := ParseGrapesFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := grapes.Name, "shell"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := len(grapes.Imports), 2; got != want {
		t.Fatalf("len(Imports) = %d, want %d", got, want)
	}

	first := grapes.Imports[0]
	if got, want := first.Import, "path"; got != want {
		t.Fatalf("Imports[0].Import = %q, want %q", got, want)
	}
	if got, want := first.Label, "path"; got != want {
		t.Fatalf("Imports[0].Label = %q, want %q", got, want)
	}
	if got, want := first.Key, "path.grape"; got != want {
		t.Fatalf("Imports[0].Key = %q, want %q", got, want)
	}
	if got, want := first.ResolvedPath, filepath.Join(dir, "path.grape"); got != want {
		t.Fatalf("Imports[0].ResolvedPath = %q, want %q", got, want)
	}

	second := grapes.Imports[1]
	if got, want := second.From, "shared"; got != want {
		t.Fatalf("Imports[1].From = %q, want %q", got, want)
	}
	if got, want := second.Import, "prompt.grape"; got != want {
		t.Fatalf("Imports[1].Import = %q, want %q", got, want)
	}
	if got, want := second.Label, "shared/prompt"; got != want {
		t.Fatalf("Imports[1].Label = %q, want %q", got, want)
	}
	if got, want := second.Key, "shared/prompt.grape"; got != want {
		t.Fatalf("Imports[1].Key = %q, want %q", got, want)
	}
	if got, want := second.ResolvedPath, filepath.Join(dir, "shared", "prompt.grape"); got != want {
		t.Fatalf("Imports[1].ResolvedPath = %q, want %q", got, want)
	}
}

func TestParseGrapesFileSupportsParentRelativeImport(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, filepath.Join(dir, "masters"), "keep.txt", "")
	path := writeTempFile(t, filepath.Join(dir, "masters"), "shell.toml", `
[[grape]]
import = "../shared/prompt.grape"
`)

	grapes, err := ParseGrapesFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := grapes.Imports[0].Key, "../shared/prompt.grape"; got != want {
		t.Fatalf("Imports[0].Key = %q, want %q", got, want)
	}
	if got, want := grapes.Imports[0].Label, "../shared/prompt"; got != want {
		t.Fatalf("Imports[0].Label = %q, want %q", got, want)
	}
	if got, want := grapes.Imports[0].ResolvedPath, filepath.Join(dir, "shared", "prompt.grape"); got != want {
		t.Fatalf("Imports[0].ResolvedPath = %q, want %q", got, want)
	}
}

func TestParseGrapesFileSupportsRelativeMasterPathFromProjectRoot(t *testing.T) {
	projectDir := t.TempDir()
	writeTempFile(t, filepath.Join(projectDir, "docs"), "grapes.toml", `
[[grape]]
import = "grapes/zoxide"
`)
	writeTempFile(t, filepath.Join(projectDir, "docs", "grapes"), "zoxide.grape", "echo zoxide\n")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Errorf("restoring working directory: %v", err)
		}
	})

	grapes, err := ParseGrapesFile("./docs/grapes.toml")
	if err != nil {
		t.Fatalf("ParseGrapesFile() returned error: %v", err)
	}

	if got, want := grapes.Imports[0].ResolvedPath, filepath.Join(projectDir, "docs", "grapes", "zoxide.grape"); got != want {
		t.Fatalf("Imports[0].ResolvedPath = %q, want %q", got, want)
	}
}

func TestParseGrapesFileInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.toml", `
[[grape]]
import = "path
`)

	_, err := ParseGrapesFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapesFileRejectsMissingImport(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.toml", `
[[grape]]
from = "shared"
`)

	_, err := ParseGrapesFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapesFileNonExistent(t *testing.T) {
	_, err := ParseGrapesFile("/nonexistent/path.toml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
