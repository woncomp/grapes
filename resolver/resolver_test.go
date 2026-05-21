package resolver

import (
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func makeGrape(key, label string) *parser.GrapeFile {
	return &parser.GrapeFile{
		Name:   label,
		Label:  label,
		Key:    key,
		Blocks: []parser.Block{{Phase: "main"}},
	}
}

func TestResolveImportsInOrder(t *testing.T) {
	grapes := []*parser.GrapeFile{
		makeGrape("prompt.grape", "prompt"),
		makeGrape("shared/path.grape", "shared/path"),
		makeGrape("aliases.grape", "aliases"),
	}
	imports := []parser.GrapeImport{
		{Key: "aliases.grape", Label: "aliases"},
		{Key: "shared/path.grape", Label: "shared/path"},
		{Key: "prompt.grape", Label: "prompt"},
	}

	resolved, err := Resolve(imports, grapes)
	if err != nil {
		t.Fatal(err)
	}

	got := labelsOf(resolved)
	want := []string{"aliases", "shared/path", "prompt"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Resolve() = %v, want %v", got, want)
	}
}

func TestResolveRejectsMissingImportedGrape(t *testing.T) {
	_, err := Resolve([]parser.GrapeImport{{Key: "missing.grape", Label: "missing"}}, []*parser.GrapeFile{makeGrape("prompt.grape", "prompt")})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing fragment: missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveIncludesDuplicatesOnlyOnce(t *testing.T) {
	resolved, err := Resolve([]parser.GrapeImport{
		{Key: "prompt.grape", Label: "prompt"},
		{Key: "prompt.grape", Label: "prompt"},
	}, []*parser.GrapeFile{makeGrape("prompt.grape", "prompt")})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(resolved), 1; got != want {
		t.Fatalf("len(Resolve()) = %d, want %d", got, want)
	}
}

func labelsOf(grapes []*parser.GrapeFile) []string {
	names := make([]string, len(grapes))
	for i, grape := range grapes {
		names[i] = grape.Label
	}
	return names
}
