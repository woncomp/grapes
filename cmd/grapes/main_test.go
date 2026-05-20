package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/woncomp/grapes/shells"
)

func TestParseArgsUsesExplicitTargets(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "-t", "zsh", "--target=bash", "--nolink"}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}

	if opts.masterPath != "master.grapes" {
		t.Fatalf("masterPath = %q, want master.grapes", opts.masterPath)
	}
	if got, want := joinTargetNames(opts.targets), "zsh, bash"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
	if !opts.noLink {
		t.Fatal("noLink = false, want true")
	}
}

func TestParseArgsSupportsYesFlag(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "--yes"}, func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.assumeYes {
		t.Fatal("assumeYes = false, want true")
	}
}

func TestParseArgsSupportsYesShortFlag(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "-y"}, func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.assumeYes {
		t.Fatal("assumeYes = false, want true")
	}
	if got, want := opts.dependencyMode, dependencyModeSafe; got != want {
		t.Fatalf("dependencyMode = %q, want %q", got, want)
	}
}

func TestParseArgsSupportsDependencyMode(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "--dependency-mode=allow-warnings"}, func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := opts.dependencyMode, dependencyModeAllowWarnings; got != want {
		t.Fatalf("dependencyMode = %q, want %q", got, want)
	}
}

func TestParseArgsRejectsInvalidDependencyMode(t *testing.T) {
	_, err := parseArgs([]string{"master.grapes", "--dependency-mode=weird"}, func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid value for --dependency-mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgsRejectsYesWithConflictingDependencyMode(t *testing.T) {
	_, err := parseArgs([]string{"master.grapes", "--yes", "--dependency-mode=fail"}, func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--yes") || !strings.Contains(err.Error(), "--dependency-mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgsUsesExplicitTargetAlias(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "-t", "nu"}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := joinTargetNames(opts.targets), "nushell"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestParseArgsUsesPowerShellAlias(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "-t", "pwsh"}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := joinTargetNames(opts.targets), "powershell"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestParseArgsDefaultsToDetectedShell(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes"}, func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := joinTargetNames(opts.targets), "zsh"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestParseArgsFailsWithoutDetectableShell(t *testing.T) {
	_, err := parseArgsWithShellDetector([]string{"master.grapes"}, func(string) (string, bool) {
		return "", false
	}, func(func(string) (string, bool)) (shells.Shell, error) {
		return nil, errors.New("could not detect current shell")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not detect current shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgsRejectsUnsupportedTarget(t *testing.T) {
	_, err := parseArgs([]string{"master.grapes", "-t", "fish"}, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unsupported target "fish"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintUsageUsesNewCommandShape(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)

	usage := buf.String()
	if !strings.Contains(usage, "Usage: grapes <input> [-t shell]... [--dependency-mode mode] [--yes] [--nolink]") {
		t.Fatalf("usage did not contain review command shape: %s", usage)
	}
	if !strings.Contains(usage, "--dependency-mode") {
		t.Fatalf("usage did not document --dependency-mode: %s", usage)
	}
	if !strings.Contains(usage, "-y, --yes") {
		t.Fatalf("usage did not document --yes: %s", usage)
	}
	if strings.Contains(usage, "--lazy") {
		t.Fatalf("usage should not mention --lazy: %s", usage)
	}
}

func TestManagedOutputDirUnix(t *testing.T) {
	dir, err := managedOutputDir("linux", func(key string) (string, bool) {
		if key == "HOME" {
			return "/tmp/home", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dir, filepath.Join("/tmp/home", ".config", "grapes"); got != want {
		t.Fatalf("managedOutputDir() = %q, want %q", got, want)
	}
}

func TestManagedOutputDirWindowsUsesAppData(t *testing.T) {
	dir, err := managedOutputDir("windows", func(key string) (string, bool) {
		if key == "APPDATA" {
			return `C:\Users\me\AppData\Roaming`, true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dir, filepath.Join(`C:\Users\me\AppData\Roaming`, "grapes"); got != want {
		t.Fatalf("managedOutputDir() = %q, want %q", got, want)
	}
}

func TestRunNoLinkGeneratesOnlySelectedTargets(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: env
---
export PROMPT_ENV=1
---
phase: main
---
echo prompt
`)

	target, err := shells.Parse("zsh")
	if err != nil {
		t.Fatal(err)
	}

	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileExists(t, filepath.Join(outputDir, "zshenv"))
	assertFileExists(t, filepath.Join(outputDir, "zshrc"))
	assertFileMissing(t, filepath.Join(outputDir, "bashenv"))
	assertFileMissing(t, filepath.Join(outputDir, "bashrc"))
	assertFileMissing(t, filepath.Join(home, ".zshenv"))
	assertFileMissing(t, filepath.Join(home, ".zshrc"))
}

func TestRunNoLinkRendersNushellEnvAndPathsNatively(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: env
env:
  PROMPT_ENV: "1"
paths:
  - /tool/bin
---
echo prompt
`)

	target, err := shells.Parse("nushell")
	if err != nil {
		t.Fatal(err)
	}

	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	data, err := os.ReadFile(filepath.Join(outputDir, "nushell-env.nu"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	assertLineContainsFragments(t, content, "$env.__GRAPES_SHELL = ", "nushell")
	assertLineContainsFragments(t, content, "$env.PROMPT_ENV = ", "1")
	assertLineContainsFragments(t, content, "$env.PATH = ", "prepend", "/tool/bin")
	assertLineExcludesFragments(t, content, "PROMPT_ENV", "export ")
	assertLineExcludesFragments(t, content, "PATH", "export ")
	assertLineExcludesFragments(t, content, "__GRAPES_SHELL", "export ")
}

func TestRunNoLinkRendersPowerShellEnvAndPathsNatively(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: env
env:
  PROMPT_ENV: "1"
paths:
  - /tool/bin
---
echo prompt
`)

	target, err := shells.Parse("powershell")
	if err != nil {
		t.Fatal(err)
	}

	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	data, err := os.ReadFile(filepath.Join(outputDir, "powershell-env.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	assertLineContainsFragments(t, content, "$env:__GRAPES_SHELL = ", "powershell")
	assertLineContainsFragments(t, content, "$env:PROMPT_ENV = ", "1")
	assertLineContainsFragments(t, content, "$env:PATH = ", "/tool/bin", "$env:PATH")
	assertLineExcludesFragments(t, content, "PROMPT_ENV", "export ")
	assertLineExcludesFragments(t, content, "PATH", "export ")
	assertLineExcludesFragments(t, content, "__GRAPES_SHELL", "export ")
}

func TestRunNoLinkBuiltinsAvoidPosixSyntaxForNushell(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	createExecutable(t, binDir, "go", "echo go version go1.24.1 linux/amd64")
	createExecutable(t, binDir, "bun", "echo 1.2.0")
	createExecutable(t, binDir, "uv", "echo uv 0.7.2")
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - go
  - bun
  - nvm
  - uv
  - zoxide
  - fzf
---
`)

	target := mustParseShell(t, "nushell")
	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	envContent := mustReadFile(t, filepath.Join(outputDir, "nushell-env.nu"))
	mainContent := mustReadFile(t, filepath.Join(outputDir, "nushell-config.nu"))
	combined := envContent + "\n" + mainContent

	assertNoPosixBuiltInSyntax(t, combined)
	assertLineContainsFragments(t, envContent, "$env.GOPATH = ", "path join", "go")
	assertLineContainsFragments(t, envContent, "$env.PATH = ", "prepend", "GOPATH")
	assertLineContainsFragments(t, envContent, "$env.BUN_INSTALL = ", "path join", ".bun")
	assertLineContainsFragments(t, envContent, "$env.PATH = ", "prepend", "BUN_INSTALL")
	assertLineContainsFragments(t, envContent, "$env.PATH = ", "prepend", ".local")
	assertFileExcludes(t, combined, "nvm.sh")
	assertFileExcludes(t, combined, "bash_completion")
	assertFileExcludes(t, combined, "fzf --bash")
	assertFileExcludes(t, combined, "fzf --zsh")
	assertFileExcludes(t, combined, "generate-shell-completion nushell")
	assertFileExcludes(t, combined, "zoxide init")
}

func TestRunNoLinkBuiltinsAvoidPosixSyntaxForPowerShell(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	createExecutable(t, binDir, "go", "echo go version go1.24.1 linux/amd64")
	createExecutable(t, binDir, "bun", "echo 1.2.0")
	createExecutable(t, binDir, "uv", "echo uv 0.7.2")
	createExecutable(t, binDir, "zoxide", "echo zoxide 0.9.4")
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - go
  - bun
  - nvm
  - uv
  - zoxide
  - fzf
---
`)

	target := mustParseShell(t, "powershell")
	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	envContent := mustReadFile(t, filepath.Join(outputDir, "powershell-env.ps1"))
	mainContent := mustReadFile(t, filepath.Join(outputDir, "powershell-profile.ps1"))
	combined := envContent + "\n" + mainContent

	assertNoPosixBuiltInSyntax(t, combined)
	assertLineContainsFragments(t, envContent, "$env:GOPATH = ", "Join-Path", "go")
	assertLineContainsFragments(t, envContent, "$env:PATH = ", "Join-Path", "GOPATH")
	assertLineContainsFragments(t, envContent, "$env:BUN_INSTALL = ", "Join-Path", ".bun")
	assertLineContainsFragments(t, envContent, "$env:PATH = ", "Join-Path", "BUN_INSTALL")
	assertLineContainsFragments(t, envContent, "$env:PATH = ", ".local", "$HOME")
	assertFileExcludes(t, combined, "nvm.sh")
	assertFileExcludes(t, combined, "bash_completion")
	assertFileExcludes(t, combined, "fzf --bash")
	assertFileExcludes(t, combined, "fzf --zsh")
	assertLineContainsFragments(t, mainContent, "generate-shell-completion powershell")
	assertLineContainsFragments(t, mainContent, "Invoke-Expression", "zoxide init powershell")
}

func TestRunDependencyChecksFileDependencyRendersWhenFileExists(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	nvmDir := filepath.Join(home, ".nvm")
	if err := os.MkdirAll(nvmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nvmDir, "nvm.sh"), []byte("echo nvm"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - nvm
---
`)
	writeTempFile(t, sourceDir, "nvm.grape", `---
phase: main
depend_file:
  paths:
    - ~/.nvm/nvm.sh
---
echo nvm-fragment
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("y\n"),
		stdout:      &bytes.Buffer{},
		interactive: true,
		noLink:      true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "nvm-fragment")
}

func TestRunDependencyChecksFileDependencySkipsWhenFileMissing(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - nvm
---
`)
	writeTempFile(t, sourceDir, "nvm.grape", `---
phase: main
depend_file:
  paths:
    - ~/.nvm/nvm.sh
---
echo nvm-fragment
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("y\n"),
		stdout:      &bytes.Buffer{},
		interactive: true,
		noLink:      true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	content := mustReadFile(t, filepath.Join(outputDir, "bashrc"))
	assertFileExcludes(t, content, "nvm-fragment")
}

func TestRunDependencyChecksSafeModeSkipsWarningsAndFailures(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "oktool", "echo oktool 1.2.3")
	createExecutable(t, binDir, "warntool", "echo warntool unknown")

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - ok
  - warn
  - miss
---
`)
	writeTempFile(t, sourceDir, "ok.grape", `---
phase: main
depend_executable:
  binary: oktool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo ok-fragment
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)
	writeTempFile(t, sourceDir, "miss.grape", `---
phase: main
depend_executable:
  binary: missingtool
---
echo miss-fragment
`)

	target := mustParseShell(t, "bash")
	var stdout bytes.Buffer
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("y\n"),
		stdout:      &stdout,
		interactive: true,
		noLink:      true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	content := mustReadFile(t, filepath.Join(outputDir, "bashrc"))
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "ok-fragment")
	assertFileExcludes(t, content, "warn-fragment")
	assertFileExcludes(t, content, "miss-fragment")
	if !strings.Contains(stdout.String(), "warning") || !strings.Contains(stdout.String(), "failed") {
		t.Fatalf("stdout = %q, want dependency table", stdout.String())
	}
}

func TestRunDependencyChecksAllowWarningsModeRendersWarnings(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "warntool", "echo warntool unknown")
	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - warn
---
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("w\n"),
		stdout:      &bytes.Buffer{},
		interactive: true,
		noLink:      true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "warn-fragment")
}

func TestRunDependencyChecksDependencyModeSafeSkipsWarnings(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "warntool", "echo warntool unknown")
	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - warn
---
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:     masterPath,
		targets:        []shells.Shell{target},
		lookupEnv:      os.LookupEnv,
		goos:           runtime.GOOS,
		stdin:          strings.NewReader(""),
		stdout:         &bytes.Buffer{},
		interactive:    false,
		dependencyMode: dependencyModeSafe,
		noLink:         true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	content := mustReadFile(t, filepath.Join(outputDir, "bashrc"))
	assertFileExcludes(t, content, "warn-fragment")
}

func TestRunDependencyChecksDependencyModeAllowWarningsRendersWarnings(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "warntool", "echo warntool unknown")
	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - warn
---
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:     masterPath,
		targets:        []shells.Shell{target},
		lookupEnv:      os.LookupEnv,
		goos:           runtime.GOOS,
		stdin:          strings.NewReader(""),
		stdout:         &bytes.Buffer{},
		interactive:    false,
		dependencyMode: dependencyModeAllowWarnings,
		noLink:         true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "warn-fragment")
}

func TestRunDependencyChecksDependencyModeFailExitsOnWarnings(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "warntool", "echo warntool unknown")
	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - warn
---
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)

	target := mustParseShell(t, "bash")
	err := runWithOptions(runOptions{
		masterPath:     masterPath,
		targets:        []shells.Shell{target},
		lookupEnv:      os.LookupEnv,
		goos:           runtime.GOOS,
		stdin:          strings.NewReader(""),
		stdout:         &bytes.Buffer{},
		interactive:    false,
		dependencyMode: dependencyModeFail,
		noLink:         true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dependency check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileMissing(t, filepath.Join(outputDir, "bashrc"))
}

func TestRunDependencyChecksYesSkipsWarnings(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "warntool", "echo warntool unknown")
	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - warn
---
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &bytes.Buffer{},
		interactive: false,
		assumeYes:   true,
		noLink:      true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	content := mustReadFile(t, filepath.Join(outputDir, "bashrc"))
	assertFileExcludes(t, content, "warn-fragment")
}

func TestRunDependencyChecksFailWhenNonInteractiveWithoutYes(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	createExecutable(t, binDir, "warntool", "echo warntool unknown")
	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - warn
---
`)
	writeTempFile(t, sourceDir, "warn.grape", `---
phase: main
depend_executable:
  binary: warntool
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo warn-fragment
`)

	target := mustParseShell(t, "bash")
	err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &bytes.Buffer{},
		interactive: false,
		noLink:      true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLinksOnlySelectedTarget(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `echo prompt
`)

	target, err := shells.Parse("bash")
	if err != nil {
		t.Fatal(err)
	}

	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		assumeYes:  true,
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileExists(t, filepath.Join(outputDir, "bashenv"))
	assertFileExists(t, filepath.Join(outputDir, "bashrc"))
	assertFileMissing(t, filepath.Join(home, ".zshenv"))
	assertFileMissing(t, filepath.Join(home, ".zshrc"))

	assertFileContains(t, filepath.Join(home, ".bashenv"), `source "`+filepath.Join(outputDir, "bashenv")+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.Join(outputDir, "bashrc")+`"`)
}

func TestRunReviewApproveInstallsAllLinks(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `echo prompt
`)

	target := mustParseShell(t, "bash")
	var stdout bytes.Buffer
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("y\ny\n"),
		stdout:      &stdout,
		interactive: true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileContains(t, filepath.Join(home, ".bashenv"), `source "`+filepath.Join(outputDir, "bashenv")+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.Join(outputDir, "bashrc")+`"`)
}

func TestRunReviewRejectSkipsAllLinks(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `echo prompt
`)

	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("n\n"),
		stdout:      &bytes.Buffer{},
		interactive: true,
	}); err != nil {
		t.Fatal(err)
	}

	assertFileMissing(t, filepath.Join(home, ".bashenv"))
	assertFileMissing(t, filepath.Join(home, ".bashrc"))
}

func TestRunReviewSkipsPromptWhenAlreadyUpToDate(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `echo prompt
`)

	target := mustParseShell(t, "bash")
	outputDir := expectedRunOutputDir(t, home, appData)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := shells.Install(filepath.Join(home, ".bashenv"), []string{`source "` + filepath.Join(outputDir, "bashenv") + `"`}); err != nil {
		t.Fatal(err)
	}
	if err := shells.Install(filepath.Join(home, ".bashrc"), []string{`source "` + filepath.Join(outputDir, "bashrc") + `"`}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &stdout,
		interactive: false,
		assumeYes:   true,
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "Apply changes") {
		t.Fatalf("stdout unexpectedly prompted: %q", stdout.String())
	}
}

func TestRunReviewYesInstallsWithoutPrompt(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `echo prompt
`)

	target := mustParseShell(t, "bash")
	var stdout bytes.Buffer
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &stdout,
		interactive: false,
		assumeYes:   true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileContains(t, filepath.Join(home, ".bashenv"), `source "`+filepath.Join(outputDir, "bashenv")+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.Join(outputDir, "bashrc")+`"`)
	if strings.Contains(stdout.String(), "Apply changes") {
		t.Fatalf("stdout unexpectedly prompted: %q", stdout.String())
	}
}

func TestRunReviewFailsWhenPromptingNonInteractive(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - prompt
---
`)
	writeTempFile(t, sourceDir, "prompt.grape", `echo prompt
`)

	target := mustParseShell(t, "bash")
	err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &bytes.Buffer{},
		interactive: false,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func expectedRunOutputDir(t *testing.T, home, appData string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		if appData == "" {
			t.Fatal("APPDATA must be set for Windows run() tests")
		}
		return filepath.Join(appData, "grapes")
	}
	return filepath.Join(home, ".config", "grapes")
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func createExecutable(t *testing.T, dir, name, command string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\n" + command + "\n"
	if runtime.GOOS == "windows" {
		path += ".bat"
		script = "@echo off\r\n" + command + "\r\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, got err=%v", path, err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s did not contain %q; got %q", path, want, string(data))
	}
}

func mustParseShell(t *testing.T, name string) shells.Shell {
	t.Helper()
	target, err := shells.Parse(name)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertFileExcludes(t *testing.T, content, forbidden string) {
	t.Helper()
	if strings.Contains(content, forbidden) {
		t.Fatalf("content unexpectedly contained %q: %q", forbidden, content)
	}
}

func assertNoPosixBuiltInSyntax(t *testing.T, content string) {
	t.Helper()
	for _, forbidden := range []string{
		`eval "$(`,
		"source <(",
		`[ -s `,
		`${VAR:-`,
		`${GOPATH:-`,
		`${BUN_INSTALL:-`,
		`${NVM_DIR:-`,
	} {
		assertFileExcludes(t, content, forbidden)
	}
}

func assertLineContainsFragments(t *testing.T, content string, fragments ...string) {
	t.Helper()

	for _, line := range strings.Split(content, "\n") {
		matches := true
		for _, fragment := range fragments {
			if !strings.Contains(line, fragment) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}

	t.Fatalf("no line in %q contained fragments %q", content, fragments)
}

func assertLineExcludesFragments(t *testing.T, content, required, forbidden string) {
	t.Helper()

	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, required) && strings.Contains(line, forbidden) {
			t.Fatalf("line %q unexpectedly contained %q", line, forbidden)
		}
	}
}
