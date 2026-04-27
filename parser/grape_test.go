package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGrapeFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "tool.grape", `---
deps:
  - path
phase: env
env:
  TOOL_HOME: "$HOME/tool"
paths:
  - $HOME/tool/bin
---

---
phase: main
---
eval "$(tool init)"
`)

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := grape.Name, "tool"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := len(grape.Deps), 1; got != want {
		t.Fatalf("len(Deps) = %d, want %d", got, want)
	}
	if got, want := grape.Deps[0], "path"; got != want {
		t.Fatalf("Deps[0] = %q, want %q", got, want)
	}
	if got, want := len(grape.Blocks), 2; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}
	if got, want := grape.Blocks[0].Phase, "env"; got != want {
		t.Fatalf("Blocks[0].Phase = %q, want %q", got, want)
	}
	if got, want := grape.Blocks[0].Env["TOOL_HOME"], "$HOME/tool"; got != want {
		t.Fatalf("Env[TOOL_HOME] = %q, want %q", got, want)
	}
	if got, want := grape.Blocks[0].Paths[0], "$HOME/tool/bin"; got != want {
		t.Fatalf("Paths[0] = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(grape.Blocks[1].Body), `eval "$(tool init)"`; got != want {
		t.Fatalf("Body = %q, want %q", got, want)
	}
}

func TestParseGrapeFileNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "plain.grape", "export FOO=bar\n")

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(grape.Blocks), 1; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}
	if got, want := grape.Blocks[0].Phase, "main"; got != want {
		t.Fatalf("Phase = %q, want %q", got, want)
	}
	if got, want := grape.Blocks[0].Body, "export FOO=bar\n"; got != want {
		t.Fatalf("Body = %q, want %q", got, want)
	}
}

func TestParseGrapeFileDefaultPhase(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "test.grape", `---
deps: []
---
some content
`)

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := grape.Blocks[0].Phase, "main"; got != want {
		t.Fatalf("Phase = %q, want %q", got, want)
	}
}

func TestParseGrapeFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
{{not yaml
---
body
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapeFileInvalidPhase(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
phase: unknown
---
body
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid phase") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGrapeFileUnterminatedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
deps:
  - path
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapeFileNonExistent(t *testing.T) {
	_, err := ParseGrapeFile("/nonexistent/path.grape")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapeString(t *testing.T) {
	grape, err := ParseGrapeString("test", `---
deps:
  - path
phase: env
---
export FOO=bar
`, "<embedded:test>")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := grape.Name, "test"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := grape.Blocks[0].Phase, "env"; got != want {
		t.Fatalf("Phase = %q, want %q", got, want)
	}
	if got, want := grape.Deps[0], "path"; got != want {
		t.Fatalf("Deps[0] = %q, want %q", got, want)
	}
	if !strings.HasPrefix(grape.Path, "<embedded:") {
		t.Fatalf("Path = %q, want embedded path", grape.Path)
	}
}

func TestParseGrapeStringKeepsStructuredEnvAndRawBody(t *testing.T) {
	grape, err := ParseGrapeString("test", `---
phase: env
env:
  FOO: bar
paths:
  - /tool/bin
---
echo raw
`, "<embedded:test>")
	if err != nil {
		t.Fatal(err)
	}

	block := grape.Blocks[0]
	if got, want := block.Env["FOO"], "bar"; got != want {
		t.Fatalf("Env[FOO] = %q, want %q", got, want)
	}
	if got, want := block.Paths[0], "/tool/bin"; got != want {
		t.Fatalf("Paths[0] = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(block.Body), "echo raw"; got != want {
		t.Fatalf("Body = %q, want %q", got, want)
	}
}

func TestParseEmbeddedGrapeLocalOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.grape"), []byte(`---
phase: env
env:
  LOCAL: "1"
---
`), 0o644); err != nil {
		t.Fatal(err)
	}

	grape, err := ParseEmbeddedGrape(dir, "test", testEmbedFS)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := grape.Blocks[0].Env["LOCAL"], "1"; got != want {
		t.Fatalf("LOCAL = %q, want %q", got, want)
	}
	if !strings.HasPrefix(grape.Path, dir) {
		t.Fatalf("Path = %q, want local path", grape.Path)
	}
}

func TestParseEmbeddedGrapeEmbeddedFallback(t *testing.T) {
	grape, err := ParseEmbeddedGrape(t.TempDir(), "test", testEmbedFS)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := grape.Blocks[0].Env["EMBEDDED"], "1"; got != want {
		t.Fatalf("EMBEDDED = %q, want %q", got, want)
	}
	if !strings.HasPrefix(grape.Path, "<embedded:") {
		t.Fatalf("Path = %q, want embedded path", grape.Path)
	}
}

func TestParseEmbeddedGrapeNeitherExists(t *testing.T) {
	_, err := ParseEmbeddedGrape(t.TempDir(), "nonexistent", testEmbedFS)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseGrapeFileMultiBlock(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "tool.grape", `---
phase: env
env:
  TOOL_HOME: "$HOME/tool"
paths:
  - $HOME/tool/bin
---

---
phase: main
---
eval "$(tool init)"
`)

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(grape.Blocks), 2; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}
	if got := strings.TrimSpace(grape.Blocks[0].Body); got != "" {
		t.Fatalf("Blocks[0].Body = %q, want empty", got)
	}
}

func TestParseGrapeStringPathsAfterEnv(t *testing.T) {
	grape, err := ParseGrapeString("test", `---
phase: env
env:
  TOOL: "$HOME/tool"
paths:
  - $TOOL/bin
---
`, "<embedded:test>")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := grape.Blocks[0].Env["TOOL"], "$HOME/tool"; got != want {
		t.Fatalf("TOOL = %q, want %q", got, want)
	}
	if got, want := grape.Blocks[0].Paths[0], "$TOOL/bin"; got != want {
		t.Fatalf("Paths[0] = %q, want %q", got, want)
	}
}

func TestParseGrapeStringDepsInSecondBlockError(t *testing.T) {
	_, err := ParseGrapeString("bad", `---
phase: env
---
body

---
deps: [foo]
---
more body
`, "<embedded:bad>")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "deps not allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGrapeStringThreeBlocks(t *testing.T) {
	grape, err := ParseGrapeString("test", `---
phase: env
env:
  A: "1"
---

---
phase: main
---
body1

---
phase: main
---
body2
`, "<embedded:test>")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(grape.Blocks), 3; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}
}
