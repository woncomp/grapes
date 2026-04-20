package shells

import (
	"path/filepath"
	"testing"
)

func TestZshManagedFilename(t *testing.T) {
	shell, err := Parse("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.ManagedFilename(PhaseEnv), "zshenv"; got != want {
		t.Fatalf("ManagedFilename(env) = %q, want %q", got, want)
	}
	if got, want := shell.ManagedFilename(PhaseMain), "zshrc"; got != want {
		t.Fatalf("ManagedFilename(main) = %q, want %q", got, want)
	}
}

func TestZshLinkTargets(t *testing.T) {
	home := t.TempDir()
	outputDir := filepath.Join(home, ".config", "grapes")

	shell, err := Parse("zsh")
	if err != nil {
		t.Fatal(err)
	}
	links := shell.LinkTargets(home, outputDir)
	if len(links) != 2 {
		t.Fatalf("len(links) = %d, want 2", len(links))
	}
	if got, want := links[0].RCFile, filepath.Join(home, ".zshenv"); got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[1].SourcePath, filepath.Join(outputDir, "zshrc"); got != want {
		t.Fatalf("links[1].SourcePath = %q, want %q", got, want)
	}
}
