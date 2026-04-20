package shells

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBashManagedFilename(t *testing.T) {
	shell, err := Parse("bash")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.ManagedFilename(PhaseEnv), "bashenv"; got != want {
		t.Fatalf("ManagedFilename(env) = %q, want %q", got, want)
	}
	if got, want := shell.ManagedFilename(PhaseMain), "bashrc"; got != want {
		t.Fatalf("ManagedFilename(main) = %q, want %q", got, want)
	}
}

func TestBashLinkTargetsPreferBashProfile(t *testing.T) {
	home := t.TempDir()
	outputDir := filepath.Join(home, ".config", "grapes")
	profile := filepath.Join(home, ".bash_profile")
	if err := os.WriteFile(profile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	shell, err := Parse("bash")
	if err != nil {
		t.Fatal(err)
	}
	links := shell.LinkTargets(home, outputDir)
	if got, want := links[0].RCFile, profile; got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[1].SourcePath, filepath.Join(outputDir, "bashrc"); got != want {
		t.Fatalf("links[1].SourcePath = %q, want %q", got, want)
	}
}
