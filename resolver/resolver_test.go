package resolver

import (
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func makeGrape(name string) *parser.GrapeFile {
	return &parser.GrapeFile{
		Name:   name,
		Blocks: []parser.Block{{Phase: "main"}},
	}
}

func TestResolveImportsInOrder(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("prompt"),
		makeGrape("path"),
		makeGrape("aliases"),
	}
	imports := []string{"aliases", "path", "prompt"}

	resolved, err := Resolve(imports, grapes)
	if err != nil {
		t.Fatal(err)
	}

	got := namesOf(resolved)
	want := []string{"aliases", "path", "prompt"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Resolve() = %v, want %v", got, want)
	}
}

func TestResolveRejectsMissingImportedGrape(t *testing.T) {
	_, err := Resolve([]string{"missing"}, []*parser.GrapeFile{makeGrape("prompt")})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing fragment: missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveIncludesDuplicatesOnlyOnce(t *testing.T) {
	resolved, err := Resolve([]string{"prompt", "prompt"}, []*parser.GrapeFile{makeGrape("prompt")})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(resolved), 1; got != want {
		t.Fatalf("len(Resolve()) = %d, want %d", got, want)
	}
}

func namesOf(grapes []*parser.GrapeFile) []string {
	names := make([]string, len(grapes))
	for i, grape := range grapes {
		names[i] = grape.Name
	}
	return names
}
