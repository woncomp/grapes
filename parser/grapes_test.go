package parser

import "testing"

func TestParseGrapesFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "shell.grapes", `---
imports:
  - path
  - prompt
---
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
	if got, want := grapes.Imports[0], "path"; got != want {
		t.Fatalf("Imports[0] = %q, want %q", got, want)
	}
	if got, want := grapes.Imports[1], "prompt"; got != want {
		t.Fatalf("Imports[1] = %q, want %q", got, want)
	}
}

func TestParseGrapesFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grapes", `---
{{not yaml
---
`)

	_, err := ParseGrapesFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapesFileUnterminatedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grapes", `---
imports:
  - path
`)

	_, err := ParseGrapesFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapesFileNonExistent(t *testing.T) {
	_, err := ParseGrapesFile("/nonexistent/path.grapes")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
