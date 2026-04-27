package shells

import "testing"

func TestPowerShellManagedFilename(t *testing.T) {
	shell, err := Parse("pwsh")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.ManagedFilename(PhaseEnv), "powershell-env.ps1"; got != want {
		t.Fatalf("ManagedFilename(env) = %q, want %q", got, want)
	}
	if got, want := shell.ManagedFilename(PhaseMain), "powershell-profile.ps1"; got != want {
		t.Fatalf("ManagedFilename(main) = %q, want %q", got, want)
	}
}

func TestPowerShellLinkTargetsUnix(t *testing.T) {
	shell, err := Parse("powershell")
	if err != nil {
		t.Fatal(err)
	}

	links, err := shell.LinkTargets(TargetContext{
		GOOS: "linux",
		LookupEnv: func(key string) (string, bool) {
			if key == "HOME" {
				return "/tmp/home", true
			}
			return "", false
		},
		OutputDir: "/tmp/home/.config/grapes",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 {
		t.Fatalf("len(links) = %d, want 1", len(links))
	}
	if got, want := links[0].RCFile, "/tmp/home/.config/powershell/Microsoft.PowerShell_profile.ps1"; got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := len(links[0].InstallLines), 2; got != want {
		t.Fatalf("len(links[0].InstallLines) = %d, want %d", got, want)
	}
	if got, want := links[0].InstallLines[0], `. "/tmp/home/.config/grapes/powershell-env.ps1"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[1], `. "/tmp/home/.config/grapes/powershell-profile.ps1"`; got != want {
		t.Fatalf("links[0].InstallLines[1] = %q, want %q", got, want)
	}
}

func TestPowerShellLinkTargetsWindows(t *testing.T) {
	shell, err := Parse("powershell")
	if err != nil {
		t.Fatal(err)
	}

	links, err := shell.LinkTargets(TargetContext{
		GOOS: "windows",
		LookupEnv: func(key string) (string, bool) {
			if key == "USERPROFILE" {
				return `C:\Users\me`, true
			}
			return "", false
		},
		OutputDir: `C:\Users\me\AppData\Roaming\grapes`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 {
		t.Fatalf("len(links) = %d, want 1", len(links))
	}
	if got, want := links[0].RCFile, `C:\Users\me\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`; got != want {
		t.Fatalf("links[0].RCFile = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[0], `. "C:\Users\me\AppData\Roaming\grapes\powershell-env.ps1"`; got != want {
		t.Fatalf("links[0].InstallLines[0] = %q, want %q", got, want)
	}
	if got, want := links[0].InstallLines[1], `. "C:\Users\me\AppData\Roaming\grapes\powershell-profile.ps1"`; got != want {
		t.Fatalf("links[0].InstallLines[1] = %q, want %q", got, want)
	}
}
