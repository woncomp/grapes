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
		Blocks: []parser.Block{{Phase: "main"}},
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

func TestThreeNodeCycle(t *testing.T) {
	fragments := []*parser.Fragment{
		makeFrag("a", "b"),
		makeFrag("b", "c"),
		makeFrag("c", "a"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, fragments)
	if err == nil {
		t.Fatal("expected cycle error for 3-node cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention circular dependency, got: %s", err.Error())
	}
}

func TestCycleDetectionWithBacktracking(t *testing.T) {
	// a -> x, a -> y, y -> y (self-cycle)
	// DFS from a explores x first (no cycle), backtracks, then finds y -> y
	fragments := []*parser.Fragment{
		makeFrag("a", "x", "y"),
		makeFrag("x"),
		makeFrag("y", "y"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, fragments)
	if err == nil {
		t.Fatal("expected cycle error with backtracking, got nil")
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
