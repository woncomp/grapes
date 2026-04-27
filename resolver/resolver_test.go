package resolver

import (
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func makeGrape(name string, deps ...string) *parser.GrapeFile {
	return &parser.GrapeFile{
		Name:   name,
		Deps:   deps,
		Blocks: []parser.Block{{Phase: "main"}},
	}
}

func TestSimpleOrder(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("aliases"),
		makeGrape("path"),
		makeGrape("prompt"),
	}
	imports := []string{"aliases", "path", "prompt"}

	sorted, err := Resolve(imports, grapes)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	if len(names) != 3 {
		t.Fatalf("got %d grapes, want 3", len(names))
	}
	for _, want := range []string{"aliases", "path", "prompt"} {
		if !contains(names, want) {
			t.Errorf("missing %s in sorted result", want)
		}
	}
}

func TestDependencyOrder(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("completions", "path"),
		makeGrape("path"),
		makeGrape("prompt"),
	}
	imports := []string{"completions", "path", "prompt"}

	sorted, err := Resolve(imports, grapes)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	pathIdx := indexOf(names, "path")
	completionsIdx := indexOf(names, "completions")

	if pathIdx == -1 || completionsIdx == -1 {
		t.Fatal("missing expected grapes")
	}
	if pathIdx > completionsIdx {
		t.Errorf("path (index %d) should come before completions (index %d)", pathIdx, completionsIdx)
	}
}

func TestTransitiveDeps(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("completions", "path"),
		makeGrape("path", "env"),
		makeGrape("env"),
	}
	imports := []string{"completions"}

	sorted, err := Resolve(imports, grapes)
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
	grapes := []*parser.GrapeFile{
		makeGrape("a", "b"),
		makeGrape("b", "a"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, grapes)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention circular dependency, got: %s", err.Error())
	}
}

func TestThreeNodeCycle(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("a", "b"),
		makeGrape("b", "c"),
		makeGrape("c", "a"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, grapes)
	if err == nil {
		t.Fatal("expected cycle error for 3-node cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention circular dependency, got: %s", err.Error())
	}
}

func TestCycleDetectionWithBacktracking(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("a", "x", "y"),
		makeGrape("x"),
		makeGrape("y", "y"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, grapes)
	if err == nil {
		t.Fatal("expected cycle error with backtracking, got nil")
	}
}

func TestMissingDependency(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("a", "nonexistent"),
	}
	imports := []string{"a"}

	_, err := Resolve(imports, grapes)
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}
}

func TestUnreachableIgnored(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("used"),
		makeGrape("unused"),
	}
	imports := []string{"used"}

	sorted, err := Resolve(imports, grapes)
	if err != nil {
		t.Fatal(err)
	}

	names := namesOf(sorted)
	if contains(names, "unused") {
		t.Error("unreachable grape 'unused' should not appear in result")
	}
}

func namesOf(grapes []*parser.GrapeFile) []string {
	names := make([]string, len(grapes))
	for i, grape := range grapes {
		names[i] = grape.Name
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
