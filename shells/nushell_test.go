package shells

import (
	"path/filepath"
	"testing"
)

func TestNushellManagedFilename(t *testing.T) {
	shell, err := Parse("nu")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.ManagedFilename(PhaseEnv), "nushell-env.nu"; got != want {
		t.Fatalf("ManagedFilename(env) = %q, want %q", got, want)
	}
	if got, want := shell.ManagedFilename(PhaseMain), "nushell-config.nu"; got != want {
		t.Fatalf("ManagedFilename(main) = %q, want %q", got, want)
	}
}

func TestNushellLinkTargetsWindows(t *testing.T) {
	shell, err := Parse("nushell")
	if err != nil {
		t.Fatal(err)
	}

	links, err := shell.LinkTargets(TargetContext{
		GOOS: "windows",
		LookupEnv: func(key string) (string, bool) {
			if key == "APPDATA" {
				return `C:\Users\me\AppData\Roaming`, true
			}
			return "", false
		},
		OutputDir: `C:\Users\me\AppData\Roaming\grapes`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(links), 2; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}

	if got, want := links[0].RCFile, filepath.Join(`C:\Users\me\AppData\Roaming`, "nushell", "env.nu"); got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[0], "source-env `C:\\Users\\me\\AppData\\Roaming\\grapes\\nushell-env.nu`"; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[1].RCFile, filepath.Join(`C:\Users\me\AppData\Roaming`, "nushell", "config.nu"); got != want {
		t.Fatalf("links[1].RCFile = %q, want %q", got, want)
	}
	if got, want := links[1].InstallLines[0], "source `C:\\Users\\me\\AppData\\Roaming\\grapes\\nushell-config.nu`"; got != want {
		t.Fatalf("links[1].InstallLines[0] = %q, want %q", got, want)
	}
}

func TestNushellLinkTargetsUnix(t *testing.T) {
	shell, err := Parse("nushell")
	if err != nil {
		t.Fatal(err)
	}

	links, err := shell.LinkTargets(TargetContext{
		GOOS: "linux",
		LookupEnv: func(key string) (string, bool) {
			if key == "HOME" {
				return "/home/me", true
			}
			return "", false
		},
		OutputDir: "/home/me/.config/grapes",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := links[0].RCFile, filepath.Join("/home/me", ".config", "nushell", "env.nu"); got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[1].RCFile, filepath.Join("/home/me", ".config", "nushell", "config.nu"); got != want {
		t.Fatalf("links[1].RCFile = %q, want %q", got, want)
	}
}
