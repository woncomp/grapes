# Shell-Native Env and Path Rendering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make frontmatter `env` and `paths` render in shell-native syntax for `bash`, `zsh`, `nushell`, and `powershell` so the new shell targets produce usable managed files.

**Architecture:** Preserve structured `Env` and `Paths` data through parsing, then add one shell-aware rendering step during `cmd/grapes/run()` after the target shell is known. Keep preprocessing and writing unchanged aside from consuming rendered content instead of parser-expanded POSIX strings.

**Tech Stack:** Go, standard library, existing `parser`, `cmd/grapes`, `preprocessor`, `writer`, and package-level Go tests

---

## File Structure

### Existing files to modify

- `parser/parser.go` — stop flattening `env` and `paths` into POSIX shell code during parsing; preserve raw body while keeping structured frontmatter data.
- `parser/parser_test.go` — verify parsed blocks keep `Env`/`Paths` as data and no longer inject `export ...` lines into `Body`.
- `cmd/grapes/main.go` — render shell-native block content before preprocessing each block.
- `cmd/grapes/main_test.go` — add end-to-end output assertions proving `nushell` and `powershell` managed files contain native `env`/`paths` syntax instead of POSIX exports.

### New files to create

- `renderer/renderer.go` — shell-aware `env`/`paths` rendering for all supported shells.
- `renderer/renderer_test.go` — focused rendering tests for `bash`, `zsh`, `nushell`, and `powershell`.

## Task 1: Preserve structured frontmatter data in parser output

**Files:**
- Modify: `parser/parser.go:15-191`
- Modify: `parser/parser_test.go:26-224`

- [ ] **Step 1: Write the failing parser tests**

```go
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
```

- [ ] **Step 2: Run the parser tests to verify they fail**

Run: `go test ./parser -run 'TestParseFragmentKeepsStructuredEnvAndRawBody|TestParseStringKeepsStructuredEnvAndRawBody'`

Expected: FAIL because `Body` still contains parser-expanded `export ...` lines.

- [ ] **Step 3: Remove parser-time shell expansion**

```go
block := Block{
	Phase: phase,
	Env:   parsed.Env,
	Paths: parsed.Paths,
	Body:  rb.Body,
}
```

Delete the now-unused `expandBlock(...)` helper from `parser/parser.go` and remove its `maps` / `slices` imports if they are no longer needed there.

- [ ] **Step 4: Run the parser package tests**

Run: `go test ./parser`

Expected: PASS

- [ ] **Step 5: Commit the parser change**

```bash
git add parser/parser.go parser/parser_test.go
git commit -m $'refactor(parser): preserve raw env and path frontmatter\n\nCo-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>'
```

## Task 2: Add shell-aware block rendering

**Files:**
- Create: `renderer/renderer.go`
- Create: `renderer/renderer_test.go`

- [ ] **Step 1: Write the failing renderer tests**

```go
func TestRenderBlockPowershell(t *testing.T) {
	got, err := RenderBlock("powershell", map[string]string{
		"FOO": "bar",
	}, []string{"/tool/bin"}, "echo done\n")
	if err != nil {
		t.Fatal(err)
	}

	want := "$env:FOO = \"bar\"\n$env:PATH = \"/tool/bin;$env:PATH\"\necho done\n"
	if got != want {
		t.Fatalf("RenderBlock() = %q, want %q", got, want)
	}
}

func TestRenderBlockNushell(t *testing.T) {
	got, err := RenderBlock("nushell", map[string]string{
		"FOO": "bar",
	}, []string{"/tool/bin"}, "echo done\n")
	if err != nil {
		t.Fatal(err)
	}

	want := "$env.FOO = \"bar\"\n$env.PATH = ($env.PATH | prepend \"/tool/bin\")\necho done\n"
	if got != want {
		t.Fatalf("RenderBlock() = %q, want %q", got, want)
	}
}

func TestRenderBlockPreservesSortedEnvOrder(t *testing.T) {
	got, err := RenderBlock("bash", map[string]string{
		"ZED": "2",
		"ALPHA": "1",
	}, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	want := "export ALPHA=\"1\"\nexport ZED=\"2\"\n"
	if got != want {
		t.Fatalf("RenderBlock() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the renderer tests to verify they fail**

Run: `go test ./renderer`

Expected: FAIL with `undefined: RenderBlock`

- [ ] **Step 3: Implement the shell-aware renderer**

```go
package renderer

func RenderBlock(shell string, env map[string]string, paths []string, body string) (string, error) {
	var lines []string

	for _, key := range slices.Sorted(maps.Keys(env)) {
		switch shell {
		case "bash", "zsh":
			lines = append(lines, fmt.Sprintf(`export %s="%s"`, key, env[key]))
		case "nushell":
			lines = append(lines, fmt.Sprintf(`$env.%s = "%s"`, key, env[key]))
		case "powershell":
			lines = append(lines, fmt.Sprintf(`$env:%s = "%s"`, key, env[key]))
		default:
			return "", fmt.Errorf("unsupported shell %q", shell)
		}
	}

	for _, path := range paths {
		switch shell {
		case "bash", "zsh":
			lines = append(lines, fmt.Sprintf(`export PATH="%s:$PATH"`, path))
		case "nushell":
			lines = append(lines, fmt.Sprintf(`$env.PATH = ($env.PATH | prepend "%s")`, path))
		case "powershell":
			lines = append(lines, fmt.Sprintf(`$env:PATH = "%s;$env:PATH"`, path))
		default:
			return "", fmt.Errorf("unsupported shell %q", shell)
		}
	}

	if body != "" {
		lines = append(lines, strings.TrimRight(body, "\n"))
	}
	if len(lines) == 0 {
		return "", nil
	}
	return strings.Join(lines, "\n") + "\n", nil
}
```

- [ ] **Step 4: Run the renderer package tests**

Run: `go test ./renderer`

Expected: PASS

- [ ] **Step 5: Commit the renderer package**

```bash
git add renderer/renderer.go renderer/renderer_test.go
git commit -m $'feat(renderer): add shell-native env path rendering\n\nCo-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>'
```

## Task 3: Wire rendering into CLI output generation and close the runtime gap

**Files:**
- Modify: `cmd/grapes/main.go:145-219`
- Modify: `cmd/grapes/main_test.go:142-260`
- Modify: `parser/parser_test.go:26-224` (only if a parser assertion needs updating after Task 1)
- Test: `renderer/renderer_test.go`

- [ ] **Step 1: Write the failing end-to-end shell-output tests**

```go
func TestRunNoLinkRendersNushellEnvAndPathsNatively(t *testing.T) {
	home := t.TempDir()
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: env
env:
  PROMPT_ENV: "1"
paths:
  - /tool/bin
---
echo prompt
`)

	target, err := shells.Parse("nushell")
	if err != nil {
		t.Fatal(err)
	}

	if err := run(masterPath, []shells.Shell{target}, true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".config", "grapes", "nushell-env.nu"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `$env.PROMPT_ENV = "1"`) {
		t.Fatalf("missing nushell env assignment: %q", content)
	}
	if !strings.Contains(content, `$env.PATH = ($env.PATH | prepend "/tool/bin")`) {
		t.Fatalf("missing nushell path prepend: %q", content)
	}
	if strings.Contains(content, `export PROMPT_ENV="1"`) {
		t.Fatalf("should not contain POSIX export syntax: %q", content)
	}
}

func TestRunNoLinkRendersPowerShellEnvAndPathsNatively(t *testing.T) {
	home := t.TempDir()
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: env
env:
  PROMPT_ENV: "1"
paths:
  - /tool/bin
---
echo prompt
`)

	target, err := shells.Parse("powershell")
	if err != nil {
		t.Fatal(err)
	}

	if err := run(masterPath, []shells.Shell{target}, true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".config", "grapes", "powershell-env.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `$env:PROMPT_ENV = "1"`) {
		t.Fatalf("missing powershell env assignment: %q", content)
	}
	if !strings.Contains(content, `$env:PATH = "/tool/bin;$env:PATH"`) {
		t.Fatalf("missing powershell path prepend: %q", content)
	}
	if strings.Contains(content, `export PROMPT_ENV="1"`) {
		t.Fatalf("should not contain POSIX export syntax: %q", content)
	}
}
```

- [ ] **Step 2: Run the focused end-to-end tests to verify they fail**

Run: `go test ./cmd/grapes -run 'TestRunNoLinkRendersNushellEnvAndPathsNatively|TestRunNoLinkRendersPowerShellEnvAndPathsNatively'`

Expected: FAIL because output still contains parser-expanded POSIX `export ...` lines.

- [ ] **Step 3: Render blocks per shell in `run()`**

```go
rendered, err := renderer.RenderBlock(target.Name(), block.Env, block.Paths, block.Body)
if err != nil {
	return fmt.Errorf("rendering %s for %s: %w", f.Name, target.Name(), err)
}

content, err := preprocessor.Process(rendered, target.Name())
if err != nil {
	return fmt.Errorf("preprocessing %s for %s: %w", f.Name, target.Name(), err)
}
```

Add the renderer import:

```go
"github.com/woncomp/grapes/renderer"
```

Use `rendered` for every block instead of passing `block.Body` directly into the preprocessor.

- [ ] **Step 4: Run focused packages and the full suite**

Run: `go test ./parser ./renderer ./cmd/grapes && go test ./...`

Expected: PASS

- [ ] **Step 5: Commit the integration change**

```bash
git add cmd/grapes/main.go cmd/grapes/main_test.go parser/parser.go parser/parser_test.go renderer/renderer.go renderer/renderer_test.go
git commit -m $'feat(cli): render env and paths per shell\n\nCo-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>'
```

## Self-Review Notes

### Spec coverage

- Parser stays shell-agnostic and preserves structured `Env`/`Paths` data in Task 1.
- One shell-aware rendering layer is added in Task 2.
- `bash`, `zsh`, `nushell`, and `powershell` rendering rules are covered in Task 2 tests.
- End-to-end runtime validation for `nushell` and `powershell` managed files is covered in Task 3.

### Placeholder scan

- No `TODO`, `TBD`, or deferred implementation placeholders remain.
- Each code-changing step includes exact code and commands.
- No task refers vaguely to another task without restating the necessary code.

### Type consistency

- Planned names are consistent across tasks:
  - `Block.Env`
  - `Block.Paths`
  - `renderer.RenderBlock`
  - `managedOutputDir`
  - `TargetContext`

