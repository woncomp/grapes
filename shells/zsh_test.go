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
	if got, want := shell.ManagedFilename(PhaseSetup), "zsh-setup"; got != want {
		t.Fatalf("ManagedFilename(setup) = %q, want %q", got, want)
	}
}

func TestZshLinkTargets(t *testing.T) {
	home := t.TempDir()
	outputDir := filepath.Join(home, ".config", "grapes")

	shell, err := Parse("zsh")
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
	if len(links) != 2 {
		t.Fatalf("len(links) = %d, want 2", len(links))
	}
	if got, want := links[0].RCFile, filepath.Join(home, ".zshenv"); got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[1].InstallLines[0], `source "`+filepath.Join(outputDir, "zshrc")+`"`; got != want {
		t.Fatalf("links[1].InstallLines[0] = %q, want %q", got, want)
	}
}

func TestZshLinkTargetsUsePOSIXInstallLines(t *testing.T) {
	shell, err := Parse("zsh")
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

	if got, want := links[0].InstallLines[0], `source "C:/Users/grapes/.config/grapes/zshenv"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[1].InstallLines[0], `source "C:/Users/grapes/.config/grapes/zshrc"`; got != want {
		t.Fatalf("links[1].InstallLines[0] = %q, want %q", got, want)
	}
}

func TestZshLinkTargetsRespectsZDOTDIRForZshrc(t *testing.T) {
	home := t.TempDir()
	zdotdir := filepath.Join(home, ".config", "zsh")
	outputDir := filepath.Join(home, ".config", "grapes")

	shell, err := Parse("zsh")
	if err != nil {
		t.Fatal(err)
	}
	links, err := shell.LinkTargets(TargetContext{
		GOOS: "linux",
		LookupEnv: func(key string) (string, bool) {
			switch key {
			case "HOME":
				return home, true
			case "ZDOTDIR":
				return zdotdir, true
			default:
				return "", false
			}
		},
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := links[0].RCFile, filepath.Join(home, ".zshenv"); got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[1].RCFile, filepath.Join(zdotdir, ".zshrc"); got != want {
		t.Fatalf("links[1].RCFile = %q, want %q", got, want)
	}
}
