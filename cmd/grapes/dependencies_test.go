package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func TestExecutableDependencyCheckPathLookupSuccessWithoutVersion(t *testing.T) {
	grapes := []*parser.GrapeFile{{
		Name: "zoxide",
		DependExecutable: &parser.DependExecutable{
			Binary: "zoxide",
		},
	}}

	results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{
		lookupEnv: func(string) (string, bool) { return "", false },
		lookPath: func(file string) (string, error) {
			if file == "zoxide" {
				return "/usr/bin/zoxide", nil
			}
			return "", errors.New("not found")
		},
		runCommand: func(string, ...string) (string, error) {
			t.Fatal("runCommand should not be called when no version settings are configured")
			return "", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(results), 1; got != want {
		t.Fatalf("len(results) = %d, want %d", got, want)
	}
	if got, want := results[0].Status, dependencyStatusOK; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got, want := results[0].Location, "/usr/bin/zoxide"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
	if got, want := results[0].Version, "n/a"; got != want {
		t.Fatalf("Version = %q, want %q", got, want)
	}
}

func TestExecutableDependencyCheckSearchPathExpansion(t *testing.T) {
	home := t.TempDir()
	grapes := []*parser.GrapeFile{{
		Name: "zoxide",
		DependExecutable: &parser.DependExecutable{
			Binary:      "zoxide",
			SearchPaths: []string{"~/.local/bin", "$HOME/.cargo/bin"},
		},
	}}

	var lookedIn []string
	_, err := checkGrapeDependencies(grapes, dependencyCheckOptions{
		lookupEnv: func(key string) (string, bool) {
			if key == "HOME" {
				return home, true
			}
			return "", false
		},
		lookPath: func(string) (string, error) { return "", errors.New("not found") },
		pathExists: func(path string) bool {
			lookedIn = append(lookedIn, path)
			return false
		},
		runCommand: func(string, ...string) (string, error) { return "", nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPaths := []string{
		filepath.Join(home, ".local", "bin", "zoxide"),
		filepath.Join(home, ".cargo", "bin", "zoxide"),
	}
	for _, want := range wantPaths {
		if !containsString(lookedIn, want) {
			t.Fatalf("lookedIn = %v, want to contain %q", lookedIn, want)
		}
	}
}

func TestExecutableDependencyCheckMissingBinaryFails(t *testing.T) {
	grapes := []*parser.GrapeFile{{
		Name:             "zoxide",
		DependExecutable: &parser.DependExecutable{Binary: "zoxide"},
	}}

	results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{
		lookupEnv:  func(string) (string, bool) { return "", false },
		lookPath:   func(string) (string, error) { return "", errors.New("not found") },
		pathExists: func(string) bool { return false },
		runCommand: func(string, ...string) (string, error) { return "", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := results[0].Status, dependencyStatusFailed; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got, want := results[0].Location, "not found"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
	if got, want := results[0].Version, "n/a"; got != want {
		t.Fatalf("Version = %q, want %q", got, want)
	}
}

func TestExecutableDependencyCheckVersionWarningsAndSuccess(t *testing.T) {
	base := []*parser.GrapeFile{{
		Name: "zoxide",
		DependExecutable: &parser.DependExecutable{
			Binary:       "zoxide",
			VersionArgs:  []string{"--version"},
			VersionRegex: `([0-9]+\.[0-9]+\.[0-9]+)`,
		},
	}}

	cases := []struct {
		name       string
		output     string
		runErr     error
		wantStatus dependencyStatus
		wantVer    string
	}{
		{name: "success", output: "zoxide 0.9.4", wantStatus: dependencyStatusOK, wantVer: "0.9.4"},
		{name: "command error", runErr: errors.New("boom"), wantStatus: dependencyStatusWarning, wantVer: "unknown"},
		{name: "regex miss", output: "zoxide version unknown", wantStatus: dependencyStatusWarning, wantVer: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := checkGrapeDependencies(base, dependencyCheckOptions{
				lookupEnv: func(string) (string, bool) { return "", false },
				lookPath: func(file string) (string, error) {
					return "/usr/bin/" + file, nil
				},
				runCommand: func(string, ...string) (string, error) {
					return tc.output, tc.runErr
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got, want := results[0].Status, tc.wantStatus; got != want {
				t.Fatalf("Status = %q, want %q", got, want)
			}
			if got, want := results[0].Version, tc.wantVer; got != want {
				t.Fatalf("Version = %q, want %q", got, want)
			}
		})
	}
}

func TestFileDependencyCheckMatchesExistingFile(t *testing.T) {
	dir := t.TempDir()
	match := filepath.Join(dir, "nvm.sh")
	if err := os.WriteFile(match, []byte("echo nvm"), 0o644); err != nil {
		t.Fatal(err)
	}

	grapes := []*parser.GrapeFile{{
		Name:       "nvm",
		DependFile: &parser.DependFile{Paths: []string{filepath.Join(dir, "*.sh")}},
	}}

	results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{lookupEnv: func(string) (string, bool) { return "", false }})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := results[0].Status, dependencyStatusOK; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got, want := results[0].Location, match; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
	if got, want := results[0].Version, "n/a"; got != want {
		t.Fatalf("Version = %q, want %q", got, want)
	}
}

func TestFileDependencyCheckSupportsTildeEnvAndGlobAndRejectsDirectories(t *testing.T) {
	home := t.TempDir()
	nvmHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".nvm"), 0o755); err != nil {
		t.Fatal(err)
	}
	file1 := filepath.Join(home, ".nvm", "nvm.sh")
	if err := os.WriteFile(file1, []byte("echo nvm"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nvmHome, "v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	file2 := filepath.Join(nvmHome, "v1", "nvm.exe")
	if err := os.WriteFile(file2, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nvmHome, "dir-only.exe"), 0o755); err != nil {
		t.Fatal(err)
	}

	grapes := []*parser.GrapeFile{{
		Name: "nvm",
		DependFile: &parser.DependFile{Paths: []string{
			"~/.nvm/*.sh",
			"$NVM_HOME/*/nvm.exe",
			"$NVM_HOME/dir-only.exe",
		}},
	}}

	results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{lookupEnv: func(key string) (string, bool) {
		switch key {
		case "HOME":
			return home, true
		case "NVM_HOME":
			return nvmHome, true
		default:
			return "", false
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := results[0].Status, dependencyStatusOK; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got := results[0].Location; got != file1 && got != file2 {
		t.Fatalf("Location = %q, want one of %q or %q", got, file1, file2)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) || value == want {
			return true
		}
	}
	return false
}
