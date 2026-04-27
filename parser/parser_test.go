package parser

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed test.grape
var testEmbedFS embed.FS

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
	if len(frag.Blocks) != 1 {
		t.Fatalf("Blocks = %d, want 1", len(frag.Blocks))
	}
	if frag.Blocks[0].Phase != "env" {
		t.Errorf("Phase = %q, want %q", frag.Blocks[0].Phase, "env")
	}
	if len(frag.Deps) != 1 || frag.Deps[0] != "path" {
		t.Errorf("Deps = %v, want [path]", frag.Deps)
	}
	if frag.IsMaster {
		t.Error("IsMaster = true, want false")
	}
	if !strings.Contains(frag.Blocks[0].Body, "export FOO=bar") {
		t.Errorf("Body missing expected content, got: %q", frag.Blocks[0].Body)
	}
}

func TestParseFragmentKeepsStructuredEnvAndRawBody(t *testing.T) {
	dir := t.TempDir()
	content := `---
deps:
  - path
phase: env
env:
  FOO: bar
paths:
  - /tool/bin
---
echo raw
`
	path := writeTempFile(t, dir, "test.grape", content)

	frag, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	block := frag.Blocks[0]
	if got, want := block.Env["FOO"], "bar"; got != want {
		t.Fatalf("Env[FOO] = %q, want %q", got, want)
	}
	if got, want := len(block.Paths), 1; got != want {
		t.Fatalf("len(Paths) = %d, want %d", got, want)
	}
	if got, want := block.Paths[0], "/tool/bin"; got != want {
		t.Fatalf("Paths[0] = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(block.Body), "echo raw"; got != want {
		t.Fatalf("Body = %q, want %q", got, want)
	}
	if strings.Contains(block.Body, "export FOO") {
		t.Fatalf("Body should not contain parser-expanded exports: %q", block.Body)
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

	if len(frag.Blocks) != 1 {
		t.Fatalf("Blocks = %d, want 1", len(frag.Blocks))
	}
	if frag.Blocks[0].Phase != "main" {
		t.Errorf("Phase = %q, want %q (default)", frag.Blocks[0].Phase, "main")
	}
	if frag.Blocks[0].Body != "export FOO=bar\n" {
		t.Errorf("Body = %q, want %q", frag.Blocks[0].Body, "export FOO=bar\n")
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

	if frag.Blocks[0].Phase != "main" {
		t.Errorf("Phase = %q, want %q", frag.Blocks[0].Phase, "main")
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
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/path.grape")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestParseString(t *testing.T) {
	content := `---
deps:
  - path
phase: env
---
export FOO=bar
`
	frag, err := ParseString("test", content)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Name != "test" {
		t.Errorf("Name = %q, want %q", frag.Name, "test")
	}
	if frag.Blocks[0].Phase != "env" {
		t.Errorf("Phase = %q, want %q", frag.Blocks[0].Phase, "env")
	}
	if len(frag.Deps) != 1 || frag.Deps[0] != "path" {
		t.Errorf("Deps = %v, want [path]", frag.Deps)
	}
	if !strings.Contains(frag.Blocks[0].Body, "export FOO=bar") {
		t.Errorf("Body missing expected content, got: %q", frag.Blocks[0].Body)
	}
	if !strings.HasPrefix(frag.Path, "<embedded:") {
		t.Errorf("Path = %q, want <embedded:...>", frag.Path)
	}
}

func TestParseStringKeepsStructuredEnvAndRawBody(t *testing.T) {
	content := `---
phase: env
env:
  FOO: bar
paths:
  - /tool/bin
---
echo raw
`

	frag, err := ParseString("test", content)
	if err != nil {
		t.Fatal(err)
	}

	block := frag.Blocks[0]
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

func TestParseStringNoFrontmatter(t *testing.T) {
	frag, err := ParseString("plain", "export FOO=bar\n")
	if err != nil {
		t.Fatal(err)
	}

	if frag.Blocks[0].Phase != "main" {
		t.Errorf("Phase = %q, want %q", frag.Blocks[0].Phase, "main")
	}
	if frag.Blocks[0].Body != "export FOO=bar\n" {
		t.Errorf("Body = %q, want %q", frag.Blocks[0].Body, "export FOO=bar\n")
	}
}

func TestParseStringInvalidYAML(t *testing.T) {
	content := `---
{{not yaml
---
body
`
	_, err := ParseString("bad", content)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestParseStringInvalidPhase(t *testing.T) {
	content := `---
phase: unknown
---
body
`
	_, err := ParseString("bad", content)
	if err == nil {
		t.Error("expected error for invalid phase, got nil")
	}
}

func TestParseFileOrEmbedded_LocalOverride(t *testing.T) {
	dir := t.TempDir()
	content := `---
phase: env
env:
  LOCAL: "1"
---
`
	if err := os.WriteFile(filepath.Join(dir, "test.grape"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	frag, err := ParseFileOrEmbedded(dir, "test", testEmbedFS)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Blocks[0].Env["LOCAL"] != "1" {
		t.Errorf("local file should take precedence, got env: %v", frag.Blocks[0].Env)
	}
	if !strings.HasPrefix(frag.Path, dir) {
		t.Errorf("Path should be local, got: %q", frag.Path)
	}
}

func TestParseFileOrEmbedded_EmbeddedFallback(t *testing.T) {
	dir := t.TempDir()
	frag, err := ParseFileOrEmbedded(dir, "test", testEmbedFS)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Blocks[0].Env["EMBEDDED"] != "1" {
		t.Errorf("should fall back to embedded, got env: %v", frag.Blocks[0].Env)
	}
	if !strings.HasPrefix(frag.Path, "<embedded:") {
		t.Errorf("embedded Path should be <embedded:...>, got: %q", frag.Path)
	}
}

func TestParseFileOrEmbedded_NeitherExists(t *testing.T) {
	dir := t.TempDir()
	_, err := ParseFileOrEmbedded(dir, "nonexistent", testEmbedFS)
	if err == nil {
		t.Error("expected error when neither local nor embedded file exists")
	}
}

// --- v2 multi-block tests ---

func TestParseMultiBlock(t *testing.T) {
	dir := t.TempDir()
	content := `---
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
`
	path := writeTempFile(t, dir, "tool.grape", content)

	frag, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(frag.Blocks) != 2 {
		t.Fatalf("Blocks = %d, want 2", len(frag.Blocks))
	}

	// First block: env phase with env vars and paths
	if frag.Blocks[0].Phase != "env" {
		t.Errorf("block 0 phase = %q, want %q", frag.Blocks[0].Phase, "env")
	}
	if frag.Blocks[0].Env["TOOL_HOME"] != "$HOME/tool" {
		t.Errorf("block 0 env = %v, want TOOL_HOME=$HOME/tool", frag.Blocks[0].Env)
	}
	if len(frag.Blocks[0].Paths) != 1 || frag.Blocks[0].Paths[0] != "$HOME/tool/bin" {
		t.Errorf("block 0 paths = %v, want [$HOME/tool/bin]", frag.Blocks[0].Paths)
	}
	if strings.TrimSpace(frag.Blocks[0].Body) != "" {
		t.Errorf("block 0 body = %q, want empty raw body", frag.Blocks[0].Body)
	}

	// Second block: main phase with body
	if frag.Blocks[1].Phase != "main" {
		t.Errorf("block 1 phase = %q, want %q", frag.Blocks[1].Phase, "main")
	}
	if !strings.Contains(frag.Blocks[1].Body, `eval "$(tool init)"`) {
		t.Errorf("block 1 body missing eval, got: %q", frag.Blocks[1].Body)
	}
}

func TestParseEnvExpansion(t *testing.T) {
	frag, err := ParseString("test", `---
phase: env
env:
  GOPATH: "${GOPATH:-$HOME/go}"
  GOBIN: "$GOPATH/bin"
---
`)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := frag.Blocks[0].Env["GOBIN"], "$GOPATH/bin"; got != want {
		t.Errorf("Env[GOBIN] = %q, want %q", got, want)
	}
	if got, want := frag.Blocks[0].Env["GOPATH"], "${GOPATH:-$HOME/go}"; got != want {
		t.Errorf("Env[GOPATH] = %q, want %q", got, want)
	}
	if got := frag.Blocks[0].Body; got != "" {
		t.Errorf("Body = %q, want empty raw body", got)
	}
}

func TestParsePathsExpansion(t *testing.T) {
	frag, err := ParseString("test", `---
phase: env
paths:
  - $HOME/.local/bin
  - $HOME/tool/bin
---
`)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(frag.Blocks[0].Paths), 2; got != want {
		t.Fatalf("len(Paths) = %d, want %d", got, want)
	}
	if got, want := frag.Blocks[0].Paths[0], "$HOME/.local/bin"; got != want {
		t.Errorf("Paths[0] = %q, want %q", got, want)
	}
	if got, want := frag.Blocks[0].Paths[1], "$HOME/tool/bin"; got != want {
		t.Errorf("Paths[1] = %q, want %q", got, want)
	}
	if got := frag.Blocks[0].Body; got != "" {
		t.Errorf("Body = %q, want empty raw body", got)
	}
}

func TestParsePathsAfterEnv(t *testing.T) {
	frag, err := ParseString("test", `---
phase: env
env:
  TOOL: "$HOME/tool"
paths:
  - $TOOL/bin
---
`)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := frag.Blocks[0].Env["TOOL"], "$HOME/tool"; got != want {
		t.Fatalf("Env[TOOL] = %q, want %q", got, want)
	}
	if got, want := len(frag.Blocks[0].Paths), 1; got != want {
		t.Fatalf("len(Paths) = %d, want %d", got, want)
	}
	if got, want := frag.Blocks[0].Paths[0], "$TOOL/bin"; got != want {
		t.Fatalf("Paths[0] = %q, want %q", got, want)
	}
	if got := frag.Blocks[0].Body; got != "" {
		t.Errorf("Body = %q, want empty raw body", got)
	}
}

func TestParseDepsInSecondBlock_Error(t *testing.T) {
	_, err := ParseString("bad", `---
phase: env
---
body

---
deps: [foo]
---
more body
`)
	if err == nil {
		t.Fatal("expected error for deps in second block")
	}
	if !strings.Contains(err.Error(), "deps not allowed") {
		t.Errorf("error should mention deps not allowed, got: %s", err.Error())
	}
}

func TestParseEmptyBlockBody(t *testing.T) {
	frag, err := ParseString("test", `---
phase: env
env:
  FOO: "bar"
---
`)
	if err != nil {
		t.Fatal(err)
	}

	if frag.Blocks[0].Phase != "env" {
		t.Errorf("phase = %q, want env", frag.Blocks[0].Phase)
	}
	if got, want := frag.Blocks[0].Env["FOO"], "bar"; got != want {
		t.Errorf("Env[FOO] = %q, want %q", got, want)
	}
	if frag.Blocks[0].Body != "" {
		t.Errorf("Body = %q, want empty raw body", frag.Blocks[0].Body)
	}
}

func TestParseThreeBlocks(t *testing.T) {
	frag, err := ParseString("test", `---
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
`)
	if err != nil {
		t.Fatal(err)
	}

	if len(frag.Blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(frag.Blocks))
	}
}
