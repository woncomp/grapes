package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func TestDependencyLabelsReflectType(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nvm.sh")
	if err := os.WriteFile(file, []byte("echo nvm"), 0o644); err != nil {
		t.Fatal(err)
	}
	grapes := []*parser.GrapeFile{
		{Name: "plain"},
		{Name: "nvm", DependFile: &parser.DependFile{Paths: []string{file}}},
		{Name: "zoxide", DependExecutable: &parser.DependExecutable{Binary: "zoxide"}},
	}

	results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{
		lookupEnv: func(string) (string, bool) { return "", false },
		lookPath: func(file string) (string, error) {
			if file == "zoxide" {
				return "/usr/bin/zoxide", nil
			}
			return "", errors.New("not found")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := results[0].Dependency, "n/a"; got != want {
		t.Fatalf("plain dependency = %q, want %q", got, want)
	}
	if got, want := results[1].Dependency, "file"; got != want {
		t.Fatalf("file dependency = %q, want %q", got, want)
	}
	if got, want := results[2].Dependency, "executable:zoxide"; got != want {
		t.Fatalf("executable dependency = %q, want %q", got, want)
	}
}

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

func TestExecutableDependencyCheckWindowsCommonPathsAndExtensions(t *testing.T) {
	cases := []struct {
		name       string
		envKey     string
		pathSuffix []string
	}{
		{name: "winget links", envKey: "LOCALAPPDATA", pathSuffix: []string{"Microsoft", "WinGet", "Links", "bun.exe"}},
		{name: "scoop shims", envKey: "USERPROFILE", pathSuffix: []string{"scoop", "shims", "bun.exe"}},
		{name: "chocolatey bin", envKey: "ChocolateyInstall", pathSuffix: []string{"bin", "bun.exe"}},
		{name: "appdata npm", envKey: "APPDATA", pathSuffix: []string{"npm", "bun.exe"}},
		{name: "localappdata programs", envKey: "LOCALAPPDATA", pathSuffix: []string{"Programs", "bun.exe"}},
		{name: "program files", envKey: "ProgramFiles", pathSuffix: []string{"bun.exe"}},
		{name: "program files x86", envKey: "ProgramFiles(x86)", pathSuffix: []string{"bun.exe"}},
		{name: "cargo bin", envKey: "USERPROFILE", pathSuffix: []string{".cargo", "bin", "bun.exe"}},
		{name: "dotnet tools", envKey: "USERPROFILE", pathSuffix: []string{".dotnet", "tools", "bun.exe"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			wantPath := filepath.Join(append([]string{root}, tc.pathSuffix...)...)
			grapes := []*parser.GrapeFile{{
				Name: "bun",
				DependExecutable: &parser.DependExecutable{
					Binary: "bun",
				},
			}}

			var lookedIn []string
			results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{
				goos: "windows",
				lookupEnv: func(key string) (string, bool) {
					if key == tc.envKey {
						return root, true
					}
					return "", false
				},
				lookPath: func(string) (string, error) { return "", errors.New("not found") },
				pathExists: func(path string) bool {
					lookedIn = append(lookedIn, path)
					return path == wantPath
				},
				runCommand: func(string, ...string) (string, error) { return "", nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			if got, want := results[0].Status, dependencyStatusOK; got != want {
				t.Fatalf("Status = %q, want %q", got, want)
			}
			if got, want := results[0].Location, wantPath; got != want {
				t.Fatalf("Location = %q, want %q", got, want)
			}
			if !containsString(lookedIn, wantPath) {
				t.Fatalf("lookedIn = %v, want to contain %q", lookedIn, wantPath)
			}
		})
	}
}

func TestExecutableDependencyCheckWindowsSearchPathsResolveExeExtension(t *testing.T) {
	root := t.TempDir()
	grapes := []*parser.GrapeFile{{
		Name: "uv",
		DependExecutable: &parser.DependExecutable{
			Binary:      "uv",
			SearchPaths: []string{root},
		},
	}}

	results, err := checkGrapeDependencies(grapes, dependencyCheckOptions{
		goos:      "windows",
		lookupEnv: func(string) (string, bool) { return "", false },
		lookPath:  func(string) (string, error) { return "", errors.New("not found") },
		pathExists: func(path string) bool {
			return path == filepath.Join(root, "uv.exe")
		},
		runCommand: func(string, ...string) (string, error) { return "", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := results[0].Location, filepath.Join(root, "uv.exe"); got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestExpandSearchPathsUsesUserProfileOnWindows(t *testing.T) {
	userProfile := t.TempDir()
	paths := expandSearchPaths([]string{"~/bin"}, func(key string) (string, bool) {
		if key == "USERPROFILE" {
			return userProfile, true
		}
		return "", false
	}, "windows")
	if got, want := paths[0], filepath.Join(userProfile, "bin"); got != want {
		t.Fatalf("expandSearchPaths() = %q, want %q", got, want)
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
