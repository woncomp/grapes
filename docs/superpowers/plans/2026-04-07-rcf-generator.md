# Grapes Shell RC Generator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI tool that reads `.grapes` master files + `.grape` fragment files, resolves dependencies via topological sort, preprocesses shell-specific directives, and generates per-shell rc files.

**Architecture:** Five components — Parser (YAML frontmatter + body), Dependency Resolver (topo sort with cycle detection), Preprocessor (C-style `#ifdef`), File Writer (phase/grouped output), Lazy Installer (marker-based source injection). Each component is a separate Go package with its own tests.

**Tech Stack:** Go, `gopkg.in/yaml.v3`, standard library only otherwise.

---

## File Structure

```
/home/woncomp/nohuman/grapes/
├── go.mod
├── go.sum
├── main.go                          # CLI entry point
├── parser/
│   ├── parser.go                    # YAML frontmatter + body parsing
│   └── parser_test.go
├── resolver/
│   ├── resolver.go                  # Topological sort + cycle detection
│   └── resolver_test.go
├── preprocessor/
│   ├── preprocessor.go              # #ifdef/#ifndef/#elif/#else/#endif evaluation
│   └── preprocessor_test.go
├── writer/
│   ├── writer.go                    # Group output by phase/shell, write files
│   └── writer_test.go
├── lazy/
│   ├── lazy.go                      # Marker-based source line install/remove
│   └── lazy_test.go
└── docs/superpowers/
    ├── specs/2026-04-06-rcf-generator-design.md
    └── plans/2026-04-07-rcf-generator.md  (this file)
```

---

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/woncomp/nohuman/grapes
go mod init github.com/woncomp/grapes
```

- [ ] **Step 2: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 3: Create stub main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("grapes")
}
```

- [ ] **Step 4: Verify it builds**

```bash
go build -o grapes .
./grapes
```

Expected: prints `grapes`

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "chore: initialize go module"
```

---

### Task 2: Parser — YAML Frontmatter Parsing

**Files:**
- Create: `parser/parser.go`
- Create: `parser/parser_test.go`

The parser reads a `.grape` or `.grapes` file, splits it on `---` delimiters, parses the YAML frontmatter into a struct, and returns the raw body string.

- [ ] **Step 1: Write the failing tests**

`parser/parser_test.go`:

```go
package parser

import (
	"os"
	"path/filepath"
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
	expectedBody := "export FOO=bar\n#ifdef BASH\necho bash\n#endif\necho common\n"
	_ = expectedBody // will refine after seeing actual body
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /home/woncomp/nohuman/grapes
go test ./parser/ -v
```

Expected: FAIL with "cannot find package" or "undefined: ParseFile"

- [ ] **Step 3: Write the parser implementation**

`parser/parser.go`:

```go
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Fragment represents a parsed .grape or .grapes file.
type Fragment struct {
	Name     string   // filename without extension
	Path     string   // full file path
	Phase    string   // "env" or "main"
	Deps     []string // fragment dependencies
	Imports  []string // master-only: fragments to include
	IsMaster bool     // true if this is a .grapes file
	Body     string   // raw body after frontmatter
}

// frontmatter is the YAML structure parsed from between --- delimiters.
type frontmatter struct {
	Deps    []string `yaml:"deps"`
	Phase   string   `yaml:"phase"`
	Imports []string `yaml:"imports"`
}

// ParseFile reads a .grape or .grapes file and returns a Fragment.
func ParseFile(path string) (*Fragment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	content := string(data)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	isMaster := filepath.Ext(path) == ".grapes"

	frag := &Fragment{
		Name:     name,
		Path:     path,
		Phase:    "main",
		IsMaster: isMaster,
	}

	// Split on --- delimiters for frontmatter
	body, fm, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if fm != nil {
		var parsed frontmatter
		if err := yaml.Unmarshal([]byte(*fm), &parsed); err != nil {
			return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
		}
		frag.Deps = parsed.Deps
		frag.Imports = parsed.Imports
		if parsed.Phase != "" {
			frag.Phase = parsed.Phase
		}
	}

	frag.Body = body

	return frag, nil
}

// splitFrontmatter splits content into body and optional YAML frontmatter.
// Frontmatter is delimited by --- on its own line.
func splitFrontmatter(content string) (body string, fm *string, err error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content, nil, nil
	}

	// Find closing ---
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}

	if end == -1 {
		return "", nil, fmt.Errorf("unterminated frontmatter (missing closing ---)")
	}

	frontmatterContent := strings.Join(lines[1:end], "\n")
	bodyContent := strings.Join(lines[end+1:], "\n")

	return bodyContent, &frontmatterContent, nil
}
```

- [ ] **Step 4: Fix the test body assertion**

The `TestParseFragment` test has a placeholder body assertion. Fix it:

```go
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
	expectedBody := "export FOO=bar\n#ifdef BASH\necho bash\n#endif\necho common\n"
	if !strings.Contains(frag.Body, "export FOO=bar") {
		t.Errorf("Body missing expected content, got: %q", frag.Body)
	}
	_ = expectedBody
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./parser/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add parser/
git commit -m "feat(parser): add YAML frontmatter and body parsing for .grape/.grapes files"
```

---

### Task 3: Dependency Resolver

**Files:**
- Create: `resolver/resolver.go`
- Create: `resolver/resolver_test.go`

Takes a list of fragments and their imports, builds a DAG from `deps`, topologically sorts, detects cycles.

- [ ] **Step 1: Write the failing tests**

`resolver/resolver_test.go`:

```go
package resolver

import (
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func makeFrag(name string, deps ...string) *parser.Fragment {
	return &parser.Fragment{
		Name:  name,
		Deps:  deps,
		Phase: "main",
	}
}

func TestSimpleOrder(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("aliases"),
		makeFrag("path"),
		makeFrag("prompt"),
	}
	imports := []string{"aliases", "path", "prompt"}

	sorted, err := Resolve(imports, fragments)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	if len(names) != 3 {
		t.Fatalf("got %d fragments, want 3", len(names))
	}
	// All present, order doesn't matter for independent fragments
	for _, want := range []string{"aliases", "path", "prompt"} {
		if !contains(names, want) {
			t.Errorf("missing %s in sorted result", want)
		}
	}
}

func TestDependencyOrder(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("completions", "path"),
		makeFrag("path"),
		makeFrag("prompt"),
	}
	imports := []string{"completions", "path", "prompt"}

	sorted, err := Resolve(imports, fragments)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	pathIdx := indexOf(names, "path")
	completionsIdx := indexOf(names, "completions")

	if pathIdx == -1 || completionsIdx == -1 {
		t.Fatal("missing expected fragments")
	}
	if pathIdx > completionsIdx {
		t.Errorf("path (index %d) should come before completions (index %d)", pathIdx, completionsIdx)
	}
}

func TestTransitiveDeps(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("completions", "path"),
		makeFrag("path", "env"),
		makeFrag("env"),
	}
	imports := []string{"completions"}

	sorted, err := Resolve(imports, fragments)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	envIdx := indexOf(names, "env")
	pathIdx := indexOf(names, "path")
	completionsIdx := indexOf(names, "completions")

	if envIdx > pathIdx || pathIdx > completionsIdx {
		t.Errorf("wrong order: env=%d path=%d completions=%d (want env < path < completions)", envIdx, pathIdx, completionsIdx)
	}
}

func TestCycleDetection(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("a", "b"),
		makeFrag("b", "a"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, fragments)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention circular dependency, got: %s", err.Error())
	}
}

func TestMissingDependency(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("a", "nonexistent"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, fragments)
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}
}

func TestUnreachableIgnored(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("used"),
		makeFrag("unused"),
	}
	imports := []string{"used"}

	sorted, err := Resolve(imports, fragments)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	if contains(names, "unused") {
		t.Error("unreachable fragment 'unused' should not appear in result")
	}
}

func namesOf(frags []*parser.Fragment) []string {
	names := make([]string, len(frags))
	for i, f := range frags {
		names[i] = f.Name
	}
	return names
}

func indexOf(s []string, val string) int {
	for i, v := range s {
		if v == val {
			return i
		}
	}
	return -1
}

func contains(s []string, val string) bool {
	return indexOf(s, val) != -1
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./resolver/ -v
```

Expected: FAIL with "cannot find package" or "undefined: Resolve"

- [ ] **Step 3: Write the resolver implementation**

`resolver/resolver.go`:

```go
package resolver

import (
	"fmt"
	"sort"

	"github.com/woncomp/grapes/parser"
)

// Resolve takes the import list from the master file and all known fragments,
// and returns them in topological order based on deps.
// Fragments not reachable from imports are excluded.
// Returns an error if there are cycles or missing dependencies.
func Resolve(imports []string, fragments []*parser.Fragment) ([]*parser.Fragment, error) {
	fragMap := make(map[string]*parser.Fragment, len(fragments))
	for _, f := range fragments {
		fragMap[f.Name] = f
	}

	// Collect all reachable fragments (imports + transitive deps)
	visited := make(map[string]bool)
	var collect func(name string) error
	collect = func(name string) error {
		if visited[name] {
			return nil
		}
		f, ok := fragMap[name]
		if !ok {
			return fmt.Errorf("missing fragment: %s", name)
		}
		visited[name] = true
		for _, dep := range f.Deps {
			if err := collect(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, name := range imports {
		if err := collect(name); err != nil {
			return nil, err
		}
	}

	// Build adjacency list for reachable fragments
	inDegree := make(map[string]int)
	edges := make(map[string][]string)

	for name := range visited {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
		f := fragMap[name]
		for _, dep := range f.Deps {
			edges[dep] = append(edges[dep], name)
			inDegree[name]++
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // stable order for no-dep fragments

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		next := edges[current]
		sort.Strings(next) // deterministic
		for _, n := range next {
			inDegree[n]--
			if inDegree[n] == 0 {
				queue = append(queue, n)
				sort.Strings(queue)
			}
		}
	}

	if len(sorted) != len(visited) {
		// Find the cycle
		cycle := findCycle(visited, fragMap)
		return nil, fmt.Errorf("circular dependency: %s", cycle)
	}

	result := make([]*parser.Fragment, len(sorted))
	for i, name := range sorted {
		result[i] = fragMap[name]
	}

	return result, nil
}

// findCycle returns a human-readable cycle description.
func findCycle(visited map[string]bool, fragMap map[string]*parser.Fragment) string {
	// DFS to find a cycle
	for start := range visited {
		path := []string{start}
		seen := map[string]bool{start: true}
		if dfs(start, path, seen, fragMap, &path) {
			// Trim path to just the cycle
			last := path[len(path)-1]
			cycleStart := -1
			for i, p := range path[:len(path)-1] {
				if p == last {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := path[cycleStart:]
				cycle = append(cycle, last)
				result := ""
				for i, p := range cycle {
					if i > 0 {
						result += " -> "
					}
					result += p
				}
				return result
			}
		}
	}
	return "unknown cycle"
}

func dfs(current string, path []string, seen map[string]bool, fragMap map[string]*parser.Fragment, result *[]string) bool {
	f := fragMap[current]
	for _, dep := range f.Deps {
		if seen[dep] {
			*result = append(path, dep)
			return true
		}
		seen[dep] = true
		newPath := append(path, dep)
		if dfs(dep, newPath, seen, fragMap, result) {
			return true
		}
		delete(seen, dep)
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./resolver/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add resolver/
git commit -m "feat(resolver): add topological sort with cycle detection"
```

---

### Task 4: Preprocessor

**Files:**
- Create: `preprocessor/preprocessor.go`
- Create: `preprocessor/preprocessor_test.go`

Evaluates `#ifdef SHELL` / `#ifndef SHELL` / `#elif` / `#else` / `#endif` directives for a given target shell (bash or zsh).

- [ ] **Step 1: Write the failing tests**

`preprocessor/preprocessor_test.go`:

```go
package preprocessor

import (
	"strings"
	"testing"
)

func TestNoDirectives(t *testing.T) {
	input := "export FOO=bar\necho hello\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}

func TestIfdefMatch(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#endif\necho common\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo bash\necho common\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIfdefNoMatch(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#endif\necho common\n"
	result, err := Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo common\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIfndef(t *testing.T) {
	input := "#ifndef BASH\necho not-bash\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo not-bash\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestElse(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#else\necho other\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo bash\n"
	if result != expected {
		t.Errorf("bash: got %q, want %q", result, expected)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected = "echo other\n"
	if result != expected {
		t.Errorf("zsh: got %q, want %q", result, expected)
	}
}

func TestElif(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#elif ZSH\necho zsh\n#else\necho other\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo bash\n" {
		t.Errorf("bash: got %q, want %q", result, "echo bash\n")
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo zsh\n" {
		t.Errorf("zsh: got %q, want %q", result, "echo zsh\n")
	}
}

func TestNestedDirectives(t *testing.T) {
	input := "#ifdef BASH\n#ifdef ZSH\necho both\n#else\necho bash-only\n#endif\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo bash-only\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestUnterminatedDirective(t *testing.T) {
	input := "#ifdef BASH\necho bash\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for unterminated directive")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("error should mention unterminated, got: %s", err.Error())
	}
}

func TestUnknownDirective(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#endif\n#undef FOO\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for unknown directive")
	}
}

func TestMultipleDirectives(t *testing.T) {
	input := "export PATH=/bin\n#ifdef BASH\nexport BASH_VAR=1\n#endif\n#ifdef ZSH\nexport ZSH_VAR=1\n#endif\necho done\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "export PATH=/bin\nexport BASH_VAR=1\necho done\n"
	if result != expected {
		t.Errorf("bash: got %q, want %q", result, expected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./preprocessor/ -v
```

Expected: FAIL with "cannot find package" or "undefined: Process"

- [ ] **Step 3: Write the preprocessor implementation**

`preprocessor/preprocessor.go`:

```go
package preprocessor

import (
	"fmt"
	"strings"
)

// Process evaluates preprocessor directives in body for the given shell.
// Supported directives: #ifdef, #ifndef, #elif, #else, #endif.
func Process(body string, shell string) (string, error) {
	lines := strings.Split(body, "\n")
	var output []string
	stack := []blockState{{include: true, satisfied: true}}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if isDirective(trimmed) {
			err := handleDirective(trimmed, shell, &stack, i+1)
			if err != nil {
				return "", err
			}
			continue
		}

		if currentInclude(stack) {
			output = append(output, line)
		}
	}

	if len(stack) != 1 {
		return "", fmt.Errorf("unterminated directive (unclosed #ifdef/#ifndef)")
	}

	// Join and trim trailing newline that the split introduced
	result := strings.Join(output, "\n")
	if len(output) > 0 {
		result += "\n"
	}
	return result, nil
}

type blockState struct {
	include   bool // whether content in this block should be included
	satisfied bool // whether any branch has already matched
}

func currentInclude(stack []blockState) bool {
	for _, s := range stack {
		if !s.include {
			return false
		}
	}
	return true
}

func isDirective(line string) bool {
	return strings.HasPrefix(line, "#ifdef ") ||
		strings.HasPrefix(line, "#ifndef ") ||
		strings.HasPrefix(line, "#elif ") ||
		line == "#else" ||
		line == "#endif"
}

func handleDirective(line string, shell string, stack *[]blockState, lineNum int) error {
	parts := strings.Fields(line)
	directive := parts[0]

	switch directive {
	case "#ifdef":
		if len(parts) != 2 {
			return fmt.Errorf("line %d: #ifdef requires exactly one argument", lineNum)
		}
		match := parts[1] == shell
		parentInclude := currentInclude(*stack)
		*stack = append(*stack, blockState{
			include:   parentInclude && match,
			satisfied: match,
		})

	case "#ifndef":
		if len(parts) != 2 {
			return fmt.Errorf("line %d: #ifndef requires exactly one argument", lineNum)
		}
		match := parts[1] != shell
		parentInclude := currentInclude(*stack)
		*stack = append(*stack, blockState{
			include:   parentInclude && match,
			satisfied: match,
		})

	case "#elif":
		if len(*stack) < 2 {
			return fmt.Errorf("line %d: #elif without matching #ifdef/#ifndef", lineNum)
		}
		if len(parts) != 2 {
			return fmt.Errorf("line %d: #elif requires exactly one argument", lineNum)
		}
		top := &(*stack)[len(*stack)-1]
		if top.satisfied {
			top.include = false
		} else {
			match := parts[1] == shell
			parentInclude := true
			if len(*stack) > 1 {
				parentInclude = (*stack)[len(*stack)-2].include
			}
			top.include = parentInclude && match
			top.satisfied = top.satisfied || match
		}

	case "#else":
		if len(*stack) < 2 {
			return fmt.Errorf("line %d: #else without matching #ifdef/#ifndef", lineNum)
		}
		top := &(*stack)[len(*stack)-1]
		if top.satisfied {
			top.include = false
		} else {
			parentInclude := true
			if len(*stack) > 1 {
				parentInclude = (*stack)[len(*stack)-2].include
			}
			top.include = parentInclude
			top.satisfied = true
		}

	case "#endif":
		if len(*stack) < 2 {
			return fmt.Errorf("line %d: #endif without matching #ifdef/#ifndef", lineNum)
		}
		*stack = (*stack)[:len(*stack)-1]

	default:
		return fmt.Errorf("line %d: unknown directive %q", lineNum, directive)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./preprocessor/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add preprocessor/
git commit -m "feat(preprocessor): add C-style directive evaluation (#ifdef/#ifndef/#elif/#else/#endif)"
```

---

### Task 5: File Writer

**Files:**
- Create: `writer/writer.go`
- Create: `writer/writer_test.go`

Takes preprocessed fragments grouped by phase and shell, writes output files to a target directory.

- [ ] **Step 1: Write the failing tests**

`writer/writer_test.go`:

```go
package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
)

func TestWriteBasic(t *testing.T) {
	dir := t.TempDir()

	fragments := []*parser.Fragment{
		{Name: "path", Phase: "env", Body: "export PATH=/bin\n"},
		{Name: "prompt", Phase: "main", Body: "PS1='$ '\n"},
	}

	// Preprocess each fragment for each shell
	type processedFragment struct {
		Name    string
		Phase   string
		Content string
	}

	shells := []string{"bash", "zsh"}
	var allProcessed []processedFragment
	for _, shell := range shells {
		for _, f := range fragments {
			content, err := preprocessor.Process(f.Body, shell)
			if err != nil {
				t.Fatal(err)
			}
			allProcessed = append(allProcessed, processedFragment{
				Name:    f.Name,
				Phase:   f.Phase,
				Content: content,
			})
		}
	}

	// Group by shell+phase and write
	_ = dir
	_ = allProcessed

	// Test Write function
	bashFragments := []Fragment{
		{Name: "path", Content: "export PATH=/bin\n"},
		{Name: "prompt", Content: "PS1='$ '\n"},
	}

	outputs := []ShellOutput{
		{Shell: "bash", Phase: "env", Fragments: bashFragments[:1]},
		{Shell: "bash", Phase: "main", Fragments: bashFragments[1:]},
		{Shell: "zsh", Phase: "env", Fragments: bashFragments[:1]},
		{Shell: "zsh", Phase: "main", Fragments: bashFragments[1:]},
	}

	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}

	// Check bashenv
	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("bashenv missing path content: %q", string(data))
	}

	// Check bashrc
	data, err = os.ReadFile(filepath.Join(dir, "bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PS1=") {
		t.Errorf("bashrc missing prompt content: %q", string(data))
	}

	// Check zshenv
	data, err = os.ReadFile(filepath.Join(dir, "zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("zshenv missing path content: %q", string(data))
	}

	// Check zshrc
	data, err = os.ReadFile(filepath.Join(dir, "zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PS1=") {
		t.Errorf("zshrc missing prompt content: %q", string(data))
	}
}

func TestWriteCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "subdir", "grapes")

	outputs := []ShellOutput{
		{Shell: "bash", Phase: "main", Fragments: nil},
	}

	if err := Write(target, outputs); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("expected output directory to be created")
	}
}

func TestWriteEmptyPhase(t *testing.T) {
	dir := t.TempDir()

	outputs := []ShellOutput{
		{Shell: "bash", Phase: "env", Fragments: nil},
		{Shell: "bash", Phase: "main", Fragments: []Fragment{{Name: "test", Content: "echo hi\n"}}},
	}

	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}

	// bashenv should exist but be empty (or not exist)
	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("bashenv should be empty, got %q", string(data))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./writer/ -v
```

Expected: FAIL

- [ ] **Step 3: Write the writer implementation**

`writer/writer.go`:

```go
package writer

import (
	"fmt"
	"os"
	"path/filepath"
)

// Fragment is a preprocessed fragment ready for output.
type Fragment struct {
	Name    string
	Content string
}

// ShellOutput represents all fragments for one shell+phase combination.
type ShellOutput struct {
	Shell     string // "bash" or "zsh"
	Phase     string // "env" or "main"
	Fragments []Fragment
}

// Write generates output files in the target directory.
// Creates the directory if it doesn't exist.
// Output files: {shell}{phase_suffix} where phase_suffix is "" for main, "env" for env.
func Write(targetDir string, outputs []ShellOutput) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	for _, out := range outputs {
		filename := out.Shell + phaseSuffix(out.Phase)
		path := filepath.Join(targetDir, filename)

		var content string
		for _, f := range out.Fragments {
			content += f.Content
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

func phaseSuffix(phase string) string {
	switch phase {
	case "env":
		return "env"
	default:
		return "rc"
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./writer/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add writer/
git commit -m "feat(writer): add phase-grouped file output per shell"
```

---

### Task 6: Lazy Installer

**Files:**
- Create: `lazy/lazy.go`
- Create: `lazy/lazy_test.go`

Appends or updates marker blocks in user's system rc files to source the generated files.

- [ ] **Step 1: Write the failing tests**

`lazy/lazy_test.go`:

```go
package lazy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallFresh(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte("# existing content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Install(rcFile, "$HOME/.config/grapes/bashrc")
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "# existing content") {
		t.Error("existing content was lost")
	}
	if !strings.Contains(content, "source \"$HOME/.config/grapes/bashrc\"") {
		t.Error("missing source line")
	}
	if !strings.Contains(content, markerStart) {
		t.Error("missing marker start")
	}
	if !strings.Contains(content, markerEnd) {
		t.Error("missing marker end")
	}
}

func TestInstallUpdate(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")

	// First install
	if err := os.WriteFile(rcFile, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}

	// Second install with different path
	if err := Install(rcFile, "$HOME/.config/grapes/new-bashrc"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Count(content, markerStart) != 1 {
		t.Errorf("should have exactly one marker block, found %d", strings.Count(content, markerStart))
	}
	if !strings.Contains(content, "new-bashrc") {
		t.Error("missing updated source line")
	}
	if strings.Contains(content, "old-bashrc") {
		t.Error("old source line should be removed")
	}
}

func TestInstallEmptyFile(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Install(rcFile, "$HOME/.config/grapes/bashrc")
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "source") {
		t.Error("missing source line in empty file")
	}
}

func TestUninstall(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	original := "# existing content\n"
	if err := os.WriteFile(rcFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}

	if err := Uninstall(rcFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, markerStart) {
		t.Error("marker should be removed after uninstall")
	}
	if strings.Contains(content, "source") {
		t.Error("source line should be removed after uninstall")
	}
	if !strings.Contains(content, "# existing content") {
		t.Error("existing content should be preserved after uninstall")
	}
}

func TestDetectBashProfile(t *testing.T) {
	dir := t.TempDir()

	// No files exist — should default to .bashenv
	target := DetectBashEnvTarget(dir)
	if !strings.HasSuffix(target, ".bashenv") {
		t.Errorf("expected .bashenv fallback, got %s", target)
	}

	// Create .bash_profile — should use that
	profile := filepath.Join(dir, ".bash_profile")
	if err := os.WriteFile(profile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	target = DetectBashEnvTarget(dir)
	if !strings.HasSuffix(target, ".bash_profile") {
		t.Errorf("expected .bash_profile, got %s", target)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./lazy/ -v
```

Expected: FAIL

- [ ] **Step 3: Write the lazy installer implementation**

`lazy/lazy.go`:

```go
package lazy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	markerStart = "# >>> grapes >>>"
	markerEnd   = "# <<< grapes <<<"
)

// Install adds or updates a marker block in rcFile that sources sourcePath.
// If a marker block already exists, it is replaced.
func Install(rcFile string, sourcePath string) error {
	sourceLine := fmt.Sprintf("source \"%s\"", sourcePath)
	block := markerStart + "\n" + sourceLine + "\n" + markerEnd + "\n"

	var existing string
	if data, err := os.ReadFile(rcFile); err == nil {
		existing = string(data)
	}

	// Remove existing marker block if present
	if strings.Contains(existing, markerStart) {
		existing = removeMarkerBlock(existing)
	}

	// Append the new block
	content := strings.TrimRight(existing, "\n") + "\n" + block

	return os.WriteFile(rcFile, []byte(content), 0o644)
}

// Uninstall removes the marker block from rcFile, preserving other content.
func Uninstall(rcFile string) error {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := removeMarkerBlock(string(data))
	return os.WriteFile(rcFile, []byte(content), 0o644)
}

// removeMarkerBlock removes everything between and including marker delimiters.
func removeMarkerBlock(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inBlock := false

	for _, line := range lines {
		if strings.TrimSpace(line) == markerStart {
			inBlock = true
			continue
		}
		if strings.TrimSpace(line) == markerEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			result = append(result, line)
		}
	}

	joined := strings.Join(result, "\n")
	// Clean up any resulting blank lines
	for strings.Contains(joined, "\n\n\n") {
		joined = strings.ReplaceAll(joined, "\n\n\n", "\n\n")
	}
	return joined
}

// DetectBashEnvTarget returns the path to use for the bash env source file.
// Prefers ~/.bash_profile if it exists, otherwise ~/.bashenv.
func DetectBashEnvTarget(homeDir string) string {
	profile := filepath.Join(homeDir, ".bash_profile")
	if _, err := os.Stat(profile); err == nil {
		return profile
	}
	return filepath.Join(homeDir, ".bashenv")
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./lazy/ -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add lazy/
git commit -m "feat(lazy): add marker-based source line install/uninstall for rc files"
```

---

### Task 7: CLI Entry Point

**Files:**
- Modify: `main.go`

Ties all components together: parse master, resolve all fragments, preprocess per shell, write output, optionally install lazy sourcing.

- [ ] **Step 1: Write the main.go implementation**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/woncomp/grapes/lazy"
	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
	"github.com/woncomp/grapes/resolver"
	"github.com/woncomp/grapes/writer"
)

func main() {
	lazyFlag := flag.Bool("lazy", false, "also install source lines in system rc files")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: grapes <source.grapes> [--lazy]\n\n")
		fmt.Fprintf(os.Stderr, "Generate shell rc files from .grape fragments.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	masterPath := flag.Arg(0)
	if err := run(masterPath, *lazyFlag); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(masterPath string, doLazy bool) error {
	// 1. Parse master file
	master, err := parser.ParseFile(masterPath)
	if err != nil {
		return err
	}

	if !master.IsMaster {
		return fmt.Errorf("%s is not a .grapes file", masterPath)
	}

	if len(master.Imports) == 0 {
		return fmt.Errorf("master file has no imports")
	}

	// 2. Parse all fragments in the same directory
	fragDir := filepath.Dir(masterPath)
	fragments, err := parseAllFragments(fragDir, master.Imports)
	if err != nil {
		return err
	}

	// 3. Resolve dependencies
	sorted, err := resolver.Resolve(master.Imports, fragments)
	if err != nil {
		return err
	}

	// 4. Preprocess per shell and write output
	outputDir := filepath.Join(os.Getenv("HOME"), ".config", "grapes")
	shells := []string{"bash", "zsh"}
	phases := []string{"env", "main"}

	var outputs []writer.ShellOutput
	for _, shell := range shells {
		for _, phase := range phases {
			var frags []writer.Fragment
			for _, f := range sorted {
				if f.Phase != phase {
					continue
				}
				content, err := preprocessor.Process(f.Body, shell)
				if err != nil {
					return fmt.Errorf("preprocessing %s for %s: %w", f.Name, shell, err)
				}
				frags = append(frags, writer.Fragment{
					Name:    f.Name,
					Content: content,
				})
			}
			outputs = append(outputs, writer.ShellOutput{
				Shell:     shell,
				Phase:     phase,
				Fragments: frags,
			})
		}
	}

	if err := writer.Write(outputDir, outputs); err != nil {
		return err
	}

	fmt.Printf("Generated rc files in %s\n", outputDir)

	// 5. Lazy install
	if doLazy {
		home := os.Getenv("HOME")
		if home == "" {
			return fmt.Errorf("HOME environment variable not set")
		}

		bashEnvTarget := lazy.DetectBashEnvTarget(home)
		installMap := map[string]string{
			bashEnvTarget:                    filepath.Join(outputDir, "bashenv"),
			filepath.Join(home, ".bashrc"):   filepath.Join(outputDir, "bashrc"),
			filepath.Join(home, ".zshenv"):   filepath.Join(outputDir, "zshenv"),
			filepath.Join(home, ".zshrc"):     filepath.Join(outputDir, "zshrc"),
		}

		for rcFile, sourcePath := range installMap {
			if err := lazy.Install(rcFile, sourcePath); err != nil {
				return fmt.Errorf("installing source in %s: %w", rcFile, err)
			}
			fmt.Printf("Installed source in %s\n", rcFile)
		}
	}

	return nil
}

// parseAllFragments recursively discovers and parses all .grape files
// reachable from the given import list.
func parseAllFragments(dir string, imports []string) ([]*parser.Fragment, error) {
	seen := make(map[string]bool)
	var fragments []*parser.Fragment

	var collect func(name string) error
	collect = func(name string) error {
		if seen[name] {
			return nil
		}
		seen[name] = true

		path := filepath.Join(dir, name+".grape")
		frag, err := parser.ParseFile(path)
		if err != nil {
			return err
		}
		fragments = append(fragments, frag)

		for _, dep := range frag.Deps {
			if err := collect(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, name := range imports {
		if err := collect(name); err != nil {
			return nil, err
		}
	}

	return fragments, nil
}
```

- [ ] **Step 2: Verify it builds**

```bash
go build -o grapes .
```

Expected: builds successfully

- [ ] **Step 3: Test with sample files**

Create test fixtures and run:

```bash
mkdir -p /tmp/grapes-test
cat > /tmp/grapes-test/path.grape << 'EOF'
---
deps: []
phase: env
---
export PATH="$HOME/bin:$PATH"

#ifdef BASH
export BASH_COMPLETION_DIR="/etc/bash_completion.d"
#endif

#ifdef ZSH
fpath=(/usr/local/share/zsh-completions $fpath)
#endif
EOF

cat > /tmp/grapes-test/prompt.grape << 'EOF'
---
deps: []
phase: main
---
#ifdef BASH
PS1='\u@\h:\w\$ '
#endif

#ifdef ZSH
PROMPT='%n@%m:%~%# '
#endif
EOF

cat > /tmp/grapes-test/master.grapes << 'EOF'
---
imports:
  - path
  - prompt
---
EOF

./grapes /tmp/grapes-test/master.grapes
```

Expected: generates `~/.config/grapes/bashenv`, `bashrc`, `zshenv`, `zshrc` with correct content.

- [ ] **Step 4: Verify output content**

```bash
echo "=== bashenv ===" && cat ~/.config/grapes/bashenv
echo "=== bashrc ===" && cat ~/.config/grapes/bashrc
echo "=== zshenv ===" && cat ~/.config/grapes/zshenv
echo "=== zshrc ===" && cat ~/.config/grapes/zshrc
```

Expected:
- `bashenv` contains `export PATH=...` and `BASH_COMPLETION_DIR`, not `fpath`
- `bashrc` contains `PS1=...` bash version
- `zshenv` contains `export PATH=...` and `fpath`, not `BASH_COMPLETION_DIR`
- `zshrc` contains `PROMPT=...` zsh version

- [ ] **Step 5: Test --lazy flag**

```bash
./grapes /tmp/grapes-test/master.grapes --lazy
```

Expected: prints "Installed source in ..." for bashenv, bashrc, zshenv, zshrc. Check that the marker blocks appear in the rc files.

- [ ] **Step 6: Run all tests**

```bash
go test ./... -v
```

Expected: all packages PASS

- [ ] **Step 7: Commit**

```bash
git add main.go
git commit -m "feat(cli): wire up parser, resolver, preprocessor, writer, and lazy installer"
```

---

### Task 8: Error Handling Edge Cases

**Files:**
- Modify: `parser/parser.go` (if needed)
- Modify: `main.go` (if needed)

Verify all error paths from the spec are handled.

- [ ] **Step 1: Test missing fragment file**

```bash
cat > /tmp/grapes-test/master.grapes << 'EOF'
---
imports:
  - nonexistent
---
EOF
./grapes /tmp/grapes-test/master.grapes
```

Expected: error message mentioning `nonexistent.grape` and file not found

- [ ] **Step 2: Test circular dependency**

```bash
cat > /tmp/grapes-test/a.grape << 'EOF'
---
deps:
  - b
---
export A=1
EOF
cat > /tmp/grapes-test/b.grape << 'EOF'
---
deps:
  - a
---
export B=1
EOF
cat > /tmp/grapes-test/master.grapes << 'EOF'
---
imports:
  - a
---
EOF
./grapes /tmp/grapes-test/master.grapes
```

Expected: error mentioning "circular dependency: a -> b -> a"

- [ ] **Step 3: Test invalid YAML**

```bash
cat > /tmp/grapes-test/bad.grape << 'EOF'
---
{{not yaml
---
body
EOF
cat > /tmp/grapes-test/master.grapes << 'EOF'
---
imports:
  - bad
---
EOF
./grapes /tmp/grapes-test/master.grapes
```

Expected: error mentioning invalid YAML in bad.grape

- [ ] **Step 4: Test unknown phase**

```bash
cat > /tmp/grapes-test/bad-phase.grape << 'EOF'
---
phase: unknown
---
content
EOF
cat > /tmp/grapes-test/master.grapes << 'EOF'
---
imports:
  - bad-phase
---
EOF
./grapes /tmp/grapes-test/master.grapes
```

Note: This may not error — the writer just uses whatever phase string it gets. Consider adding phase validation in the parser. If so, add a validation step.

- [ ] **Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix: validate phase values and improve error messages"
```

---

### Task 9: Add Phase Validation

**Files:**
- Modify: `parser/parser.go`
- Modify: `parser/parser_test.go`

Validate that the `phase` field is one of the allowed values.

- [ ] **Step 1: Add validation to parser**

Add to `parser/parser.go` after parsing frontmatter:

```go
// Validate phase
if frag.Phase != "env" && frag.Phase != "main" {
    return nil, fmt.Errorf("invalid phase %q in %s (must be \"env\" or \"main\")", frag.Phase, path)
}
```

- [ ] **Step 2: Add test for invalid phase**

Add to `parser/parser_test.go`:

```go
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
```

- [ ] **Step 3: Run tests**

```bash
go test ./parser/ -v
```

Expected: all PASS including the new test

- [ ] **Step 4: Commit**

```bash
git add parser/
git commit -m "feat(parser): validate phase field values"
```

---

### Task 10: Final Integration and Cleanup

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS

- [ ] **Step 2: Build and smoke test end-to-end**

```bash
go build -o grapes .
rm -rf /tmp/grapes-test ~/.config/grapes

mkdir -p /tmp/grapes-test
# Create all test fragments from spec examples
# Run and verify output
```

- [ ] **Step 3: Clean up temp test files**

```bash
rm -rf /tmp/grapes-test
```

- [ ] **Step 4: Final commit if any changes**

```bash
git add -A
git commit -m "chore: final cleanup and integration verification"
```
