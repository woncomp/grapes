package parser

import (
	"strings"
	"testing"
)

func TestParseGrapeFile(t *testing.T) {
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

	if got, want := grape.Name, "tool"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
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

func TestParseGrapeFileSetupPhase(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "setup.grape", `---
phase: setup
---
echo setup
`)

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := grape.Blocks[0].Phase, "setup"; got != want {
		t.Fatalf("Phase = %q, want %q", got, want)
	}
}

func TestParseGrapeFileUnterminatedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
phase: main
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
phase: env
---
export FOO=bar
`, "<inline:test>")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := grape.Name, "test"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := grape.Blocks[0].Phase, "env"; got != want {
		t.Fatalf("Phase = %q, want %q", got, want)
	}
	if got, want := grape.Path, "<inline:test>"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
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
`, "<inline:test>")
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
`, "<inline:test>")
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
`, "<inline:test>")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(grape.Blocks), 3; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}
}

func TestParseGrapeFileDependExecutable(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bun.grape", `---
phase: env
depend_executable:
  binary: bun
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo bun
`)

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if grape.DependExecutable == nil {
		t.Fatal("DependExecutable = nil, want config")
	}
	if got, want := grape.DependExecutable.Binary, "bun"; got != want {
		t.Fatalf("Binary = %q, want %q", got, want)
	}
	if got, want := len(grape.DependExecutable.SearchPaths), 0; got != want {
		t.Fatalf("len(SearchPaths) = %d, want %d", got, want)
	}
	if got, want := grape.DependExecutable.VersionArgs[0], "--version"; got != want {
		t.Fatalf("VersionArgs[0] = %q, want %q", got, want)
	}
	if got, want := grape.DependExecutable.VersionRegex, "([0-9]+\\.[0-9]+\\.[0-9]+)"; got != want {
		t.Fatalf("VersionRegex = %q, want %q", got, want)
	}
}

func TestParseGrapeFileDependFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "tool.grape", `---
phase: main
depend_file:
  paths:
    - ~/.tool/tool.sh
    - $TOOL_HOME/tool.exe
---
echo tool
`)

	grape, err := ParseGrapeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if grape.DependFile == nil {
		t.Fatal("DependFile = nil, want config")
	}
	if got, want := len(grape.DependFile.Paths), 2; got != want {
		t.Fatalf("len(Paths) = %d, want %d", got, want)
	}
	if got, want := grape.DependFile.Paths[0], "~/.tool/tool.sh"; got != want {
		t.Fatalf("Paths[0] = %q, want %q", got, want)
	}
	if got, want := grape.DependFile.Paths[1], "$TOOL_HOME/tool.exe"; got != want {
		t.Fatalf("Paths[1] = %q, want %q", got, want)
	}
}

func TestParseGrapeFileRejectsDependFileWithoutPaths(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
phase: main
depend_file: {}
---
echo bad
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "depend_file") || !strings.Contains(err.Error(), "paths") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGrapeFileRejectsDeps(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
deps:
  - path
phase: main
---
echo bad
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "deps") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGrapeFileRejectsDependExecutableWithoutBinary(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
phase: main
depend_executable:
  version_args:
    - --version
---
echo bad
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGrapeFileRejectsDependExecutableInvalidRegex(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
phase: main
depend_executable:
  binary: zoxide
  version_args:
    - --version
  version_regex: "("
---
echo bad
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "version_regex") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGrapeFileRejectsDependExecutableRegexWithoutArgs(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "bad.grape", `---
phase: main
depend_executable:
  binary: zoxide
  version_regex: "([0-9]+)"
---
echo bad
`)

	_, err := ParseGrapeFile(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "version_args") {
		t.Fatalf("unexpected error: %v", err)
	}
}
