package shells

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewReturnsCurrentAndProposedContent(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte("# existing\n# >>> grapes >>>\nsource \"/old/bashrc\"\n# <<< grapes <<<\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	review, err := Preview(rcFile, []string{`source "/new/bashrc"`})
	if err != nil {
		t.Fatal(err)
	}

	if !review.Changed {
		t.Fatal("Changed = false, want true")
	}
	if !strings.Contains(review.Current, `/old/bashrc`) {
		t.Fatalf("Current = %q, want old install line", review.Current)
	}
	if !strings.Contains(review.Proposed, `/new/bashrc`) {
		t.Fatalf("Proposed = %q, want new install line", review.Proposed)
	}
}

func TestPreviewReportsNoChangeWhenInstallMatches(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	lines := []string{`source "$HOME/.local/state/grapes/bashrc"`}
	if err := Install(rcFile, lines); err != nil {
		t.Fatal(err)
	}

	review, err := Preview(rcFile, lines)
	if err != nil {
		t.Fatal(err)
	}

	if review.Changed {
		t.Fatal("Changed = true, want false")
	}
	if got := review.Diff(); got != "" {
		t.Fatalf("Diff() = %q, want empty", got)
	}
}

func TestPreviewDiffUsesUnifiedFormat(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")

	review, err := Preview(rcFile, []string{`source "/tmp/grapes/bashrc"`})
	if err != nil {
		t.Fatal(err)
	}

	diff := review.Diff()
	for _, fragment := range []string{
		"--- " + rcFile,
		"+++ " + rcFile,
		"@@",
		"+# >>> grapes >>>",
		`+source "/tmp/grapes/bashrc"`,
		"+# <<< grapes <<<",
	} {
		if !strings.Contains(diff, fragment) {
			t.Fatalf("Diff() = %q, want fragment %q", diff, fragment)
		}
	}
}
