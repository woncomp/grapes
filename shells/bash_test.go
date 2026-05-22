package shells

import (
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
	if got, want := shell.ManagedFilename(PhaseSetup), "bash-setup"; got != want {
		t.Fatalf("ManagedFilename(setup) = %q, want %q", got, want)
	}
}

func TestBashLinkTargetsUseBashrc(t *testing.T) {
	home := t.TempDir()
	outputDir := filepath.Join(home, ".local", "state", "grapes")

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
	if got, want := len(links), 1; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}
	if got, want := links[0].RCFile, filepath.Join(home, ".bashrc"); got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[0], `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashenv"))+`"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[1], `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashrc"))+`"`; got != want {
		t.Fatalf("links[0].InstallLines[1] = %q, want %q", got, want)
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
		OutputDir: `C:\Users\grapes\.local\state\grapes`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(links), 1; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}
	if got, want := links[0].RCFile, `C:\Users\grapes\.bashrc`; got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[0], `source "C:/Users/grapes/.local/state/grapes/bashenv"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[1], `source "C:/Users/grapes/.local/state/grapes/bashrc"`; got != want {
		t.Fatalf("links[0].InstallLines[1] = %q, want %q", got, want)
	}
}
