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
	links, err := shell.LinkTargets(TargetContext{
		GOOS: "linux",
		LookupEnv: func(key string) (string, bool) {
			if key == "HOME" {
				return home, true
			}
			return "", false
		},
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := links[0].RCFile, profile; got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[0], `source "`+filepath.Join(outputDir, "bashenv")+`"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[1].InstallLines[0], `source "`+filepath.Join(outputDir, "bashrc")+`"`; got != want {
		t.Fatalf("links[1].InstallLines[0] = %q, want %q", got, want)
	}
}

func TestBashLinkTargetsUsePOSIXInstallLines(t *testing.T) {
	shell, err := Parse("bash")
	if err != nil {
		t.Fatal(err)
	}

	links, err := shell.LinkTargets(TargetContext{
		GOOS: "windows",
		LookupEnv: func(key string) (string, bool) {
			if key == "USERPROFILE" {
				return `C:\Users\grapes`, true
			}
			return "", false
		},
		OutputDir: `C:\Users\grapes\.config\grapes`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := links[0].InstallLines[0], `source "C:/Users/grapes/.config/grapes/bashenv"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[1].InstallLines[0], `source "C:/Users/grapes/.config/grapes/bashrc"`; got != want {
		t.Fatalf("links[1].InstallLines[0] = %q, want %q", got, want)
	}
}
