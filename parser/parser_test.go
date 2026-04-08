package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseFragment(t *testing.T) {
	dir := t.TempDir()
	content := `---
deps:
  - path
phase: env
---
export FOO=bar
#ifdef BASH
echo bash
#endif
`
	path := writeTempFile(t, dir, "test.grape", content)

	frag, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Name != "test" {
		t.Errorf("Name = %q, want %q", frag.Name, "test")
	}
	if frag.Phase != "env" {
		t.Errorf("Phase = %q, want %q", frag.Phase, "env")
	}
	if len(frag.Deps) != 1 || frag.Deps[0] != "path" {
		t.Errorf("Deps = %v, want [path]", frag.Deps)
	}
	if frag.IsMaster {
		t.Error("IsMaster = true, want false")
	}
	if !strings.Contains(frag.Body, "export FOO=bar") {
		t.Errorf("Body missing expected content, got: %q", frag.Body)
	}
}

func TestParseMaster(t *testing.T) {
	dir := t.TempDir()
	content := `---
imports:
  - path
  - prompt
---
`
	path := writeTempFile(t, dir, "master.grapes", content)

	frag, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !frag.IsMaster {
		t.Error("IsMaster = false, want true")
	}
	if len(frag.Imports) != 2 {
		t.Errorf("Imports = %v, want [path, prompt]", frag.Imports)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "export FOO=bar\n"
	path := writeTempFile(t, dir, "plain.grape", content)

	frag, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Phase != "main" {
		t.Errorf("Phase = %q, want %q (default)", frag.Phase, "main")
	}
	if frag.Body != "export FOO=bar\n" {
		t.Errorf("Body = %q, want %q", frag.Body, "export FOO=bar\n")
	}
}

func TestParseDefaultPhase(t *testing.T) {
	dir := t.TempDir()
	content := `---
deps: []
---
some content
`
	path := writeTempFile(t, dir, "test.grape", content)

	frag, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Phase != "main" {
		t.Errorf("Phase = %q, want %q", frag.Phase, "main")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := `---
{{not yaml
---
body
`
	path := writeTempFile(t, dir, "bad.grape", content)

	_, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestParseInvalidPhase(t *testing.T) {
	dir := t.TempDir()
	content := `---
phase: unknown
---
body
`
	path := writeTempFile(t, dir, "bad.grape", content)

	_, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for invalid phase, got nil")
	}
	if !strings.Contains(err.Error(), "invalid phase") {
		t.Errorf("error should mention invalid phase, got: %s", err.Error())
	}
}

func TestParseUnterminatedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
deps:
  - path
`
	path := writeTempFile(t, dir, "bad.grape", content)

	_, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for unterminated frontmatter, got nil")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("error should mention unterminated, got: %s", err.Error())
	}
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/path.grape")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
