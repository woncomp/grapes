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

func init() {
	defaultExecuteSetup = func(shells.Shell, string) error {
		return nil
	}
}

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

func TestParseArgsUsesPwshTarget(t *testing.T) {
	opts, err := parseArgs([]string{"master.grapes", "-t", "pwsh"}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := joinTargetNames(opts.targets), "pwsh"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestParseArgsRejectsLegacyWindowsPSTargetName(t *testing.T) {
	_, err := parseArgs([]string{"master.grapes", "-t", "powershell"}, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unsupported target "powershell"`) {
		t.Fatalf("unexpected error: %v", err)
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
	assertFileMissing(t, filepath.Join(outputDir, "bash-setup"))
	assertFileMissing(t, filepath.Join(home, ".zshenv"))
	assertFileMissing(t, filepath.Join(home, ".zshrc"))

	envContent := mustReadFile(t, filepath.Join(outputDir, "zshenv"))
	mainContent := mustReadFile(t, filepath.Join(outputDir, "zshrc"))
	if !strings.Contains(envContent, "# ==== grape: prompt") {
		t.Fatalf("zshenv missing grape divider: %q", envContent)
	}
	if !strings.Contains(mainContent, "# ==== grape: prompt") {
		t.Fatalf("zshrc missing grape divider: %q", mainContent)
	}
	assertFileExcludes(t, envContent, "# ==== grape: __GRAPE_ENV")
}

func TestRunNoLinkPreservesPreviouslyGeneratedOtherShellOutputs(t *testing.T) {
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

	zsh := mustParseShell(t, "zsh")
	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{zsh},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &bytes.Buffer{},
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	bash := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{bash},
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
	assertFileExists(t, filepath.Join(outputDir, "bashenv"))
	assertFileExists(t, filepath.Join(outputDir, "bashrc"))
}

func TestRunExecutesSetupPhaseOnceWithoutLinking(t *testing.T) {
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
  - setup
---
`)
	writeTempFile(t, sourceDir, "setup.grape", `---
phase: setup
---
echo setup-fragment
`)

	var executed []string
	var stdout bytes.Buffer
	target := mustParseShell(t, "bash")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &stdout,
		interactive: false,
		assumeYes:   true,
		noLink:      true,
		executeSetup: func(shell shells.Shell, path string) error {
			executed = append(executed, shell.Name()+"|"+path)
			content := mustReadFile(t, path)
			if !strings.Contains(content, "echo setup-fragment") {
				t.Fatalf("setup file missing setup fragment: %q", content)
			}
			if !strings.Contains(content, "GRAPES_OUTPUT_PATH") {
				t.Fatalf("setup file missing injected globals: %q", content)
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	setupPath := filepath.Join(outputDir, "bash-setup")
	assertFileExists(t, setupPath)
	assertFileMissing(t, filepath.Join(home, ".bashenv"))
	assertFileMissing(t, filepath.Join(home, ".bashrc"))
	if got, want := len(executed), 1; got != want {
		t.Fatalf("len(executed) = %d, want %d", got, want)
	}
	if got, want := executed[0], "bash|"+setupPath; got != want {
		t.Fatalf("executed[0] = %q, want %q", got, want)
	}
	if !strings.Contains(stdout.String(), "Executed setup file "+setupPath) {
		t.Fatalf("stdout missing setup execution message: %q", stdout.String())
	}
}

func TestRunNoLinkReportsGeneratedFilesWithFullPaths(t *testing.T) {
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

	target := mustParseShell(t, "zsh")
	var stdout bytes.Buffer
	if err := runWithOptions(runOptions{
		masterPath: masterPath,
		targets:    []shells.Shell{target},
		lookupEnv:  os.LookupEnv,
		goos:       runtime.GOOS,
		stdin:      strings.NewReader(""),
		stdout:     &stdout,
		assumeYes:  true,
		noLink:     true,
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	text := stdout.String()
	for _, fragment := range []string{
		"Generated files:",
		filepath.Join(outputDir, "zshenv"),
		filepath.Join(outputDir, "zshrc"),
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("stdout = %q, want fragment %q", text, fragment)
		}
	}
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
	assertLineContainsFragments(t, content, "$env.GRAPES_SHELL = ", "nushell")
	assertLineContainsFragments(t, content, "$env.GRAPES_OUTPUT_PATH = ", expectedInjectedOutputPath("nushell", outputDir))
	assertLineContainsFragments(t, content, "$env.PROMPT_ENV = ", "1")
	assertLineContainsFragments(t, content, "$env.PATH = ", "prepend", "/tool/bin")
	assertLineExcludesFragments(t, content, "PROMPT_ENV", "export ")
	assertLineExcludesFragments(t, content, "PATH", "export ")
	assertLineExcludesFragments(t, content, "GRAPES_SHELL", "export ")
}

func TestRunNoLinkRendersPwshEnvAndPathsNatively(t *testing.T) {
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

	target, err := shells.Parse("pwsh")
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
	assertFileExists(t, filepath.Join(outputDir, "cache"))
	assertFileExists(t, expectedRunStateDir(home))
	data, err := os.ReadFile(filepath.Join(outputDir, "pwsh-env.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	assertLineContainsFragments(t, content, "$env:GRAPES_SHELL = ", "pwsh")
	assertLineContainsFragments(t, content, "$env:GRAPES_OUTPUT_PATH = ", expectedInjectedOutputPath("pwsh", outputDir))
	assertLineContainsFragments(t, content, "$env:GRAPES_OUT_CACHE_DIR = ", "cache")
	assertLineContainsFragments(t, content, "$env:PROMPT_ENV = ", "1")
	assertLineContainsFragments(t, content, "$env:PATH = ", "/tool/bin", "$env:PATH")
	assertLineExcludesFragments(t, content, "PROMPT_ENV", "export ")
	assertLineExcludesFragments(t, content, "PATH", "export ")
	assertLineExcludesFragments(t, content, "GRAPES_SHELL", "export ")
}

func TestRunEmitsGrapesShellOnlyInEnvOutput(t *testing.T) {
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
  - env-one
  - env-two
  - prompt
---
`)
	writeTempFile(t, sourceDir, "env-one.grape", `---
phase: env
---
export ONE=1
`)
	writeTempFile(t, sourceDir, "env-two.grape", `---
phase: env
---
export TWO=2
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: main
---
echo prompt
`)

	target := mustParseShell(t, "zsh")
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
	assertFileExists(t, filepath.Join(outputDir, "cache"))
	assertFileExists(t, expectedRunStateDir(home))
	envContent := mustReadFile(t, filepath.Join(outputDir, "zshenv"))
	if got, want := strings.Count(envContent, `GRAPES_SHELL="zsh"`), 1; got != want {
		t.Fatalf("env GRAPES_SHELL count = %d, want %d; content=%q", got, want, envContent)
	}
	if got, want := strings.Count(envContent, `GRAPES_OUTPUT_PATH="`+expectedInjectedOutputPath("zsh", outputDir)+`"`), 1; got != want {
		t.Fatalf("env GRAPES_OUTPUT_PATH count = %d, want %d; content=%q", got, want, envContent)
	}
	if got, want := strings.Count(envContent, `GRAPES_OUT_CACHE_DIR="`+expectedInjectedOutputPath("zsh", outputDir)+`/cache"`), 1; got != want {
		t.Fatalf("env GRAPES_OUT_CACHE_DIR count = %d, want %d; content=%q", got, want, envContent)
	}
	mainContent := mustReadFile(t, filepath.Join(outputDir, "zshrc"))
	if strings.Contains(mainContent, "GRAPES_SHELL") {
		t.Fatalf("zshrc unexpectedly contained GRAPES_SHELL: %q", mainContent)
	}
	if strings.Contains(mainContent, "GRAPES_OUTPUT_PATH") {
		t.Fatalf("zshrc unexpectedly contained GRAPES_OUTPUT_PATH: %q", mainContent)
	}
}

func TestRunNoLinkExampleFragmentsAvoidPosixSyntaxForNushell(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	createExecutable(t, binDir, "go", "echo go version go1.24.1 linux/amd64")
	createExecutable(t, binDir, "bun", "echo 1.2.0")
	createExecutable(t, binDir, "fnm", "echo 1.39.0")
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
  - fnm
  - uv
  - zoxide
  - fzf
---
`)
	copyExampleFragments(t, sourceDir, "go", "bun", "fnm", "uv", "zoxide", "fzf")

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
	assertFileExists(t, filepath.Join(outputDir, "cache"))
	assertFileExists(t, expectedRunStateDir(home))
	envContent := mustReadFile(t, filepath.Join(outputDir, "nushell-env.nu"))
	mainContent := mustReadFile(t, filepath.Join(outputDir, "nushell-config.nu"))
	combined := envContent + "\n" + mainContent

	assertNoPosixBuiltInSyntax(t, combined)
	assertLineContainsFragments(t, envContent, "$env.GOPATH = ", "path join", "go")
	assertLineContainsFragments(t, envContent, "$env.PATH = ", "prepend", "GOPATH")
	assertLineContainsFragments(t, envContent, "$env.BUN_INSTALL = ", "path join", ".bun")
	assertLineContainsFragments(t, envContent, "$env.PATH = ", "prepend", "BUN_INSTALL")
	assertLineContainsFragments(t, envContent, "$env.GRAPES_OUT_CACHE_DIR = ", "path join", "cache")
	assertLineContainsFragments(t, envContent, "$env.PATH = ", "prepend", "GRAPES_EXEC_DIR")
	assertLineContainsFragments(t, envContent, "$env.GRAPES_EXEC_PATH = ", "fnm")
	assertLineContainsFragments(t, envContent, "$env.GRAPES_EXEC_VERSION = ", "1.39.0")
	assertLineContainsFragments(t, envContent, "fnm env --json", "from json")
	assertFileExcludes(t, mainContent, "fnm env")
	assertFileExcludes(t, mainContent, "from json")
	assertFileExcludes(t, mainContent, "load-env")
	assertFileExcludes(t, mainContent, "FNM_MULTISHELL_PATH")
	assertFileExcludes(t, mainContent, "FNM_PATH")
	assertFileExcludes(t, combined, "fzf --bash")
	assertFileExcludes(t, combined, "fzf --zsh")
	assertFileExcludes(t, combined, "generate-shell-completion nushell")
	assertLineContainsFragments(t, mainContent, "source ~/.local/state/grapes/zoxide.nu")
}

func TestRunNoLinkExampleFragmentsAvoidPosixSyntaxForPwsh(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	createExecutable(t, binDir, "go", "echo go version go1.24.1 linux/amd64")
	createExecutable(t, binDir, "bun", "echo 1.2.0")
	createExecutable(t, binDir, "fnm", "echo 1.39.0")
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
  - fnm
  - uv
  - zoxide
  - fzf
---
`)
	copyExampleFragments(t, sourceDir, "go", "bun", "fnm", "uv", "zoxide", "fzf")

	var executed []string
	var stdout bytes.Buffer
	target := mustParseShell(t, "pwsh")
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader(""),
		stdout:      &stdout,
		assumeYes:   true,
		noLink:      true,
		executeSetup: func(shell shells.Shell, path string) error {
			executed = append(executed, shell.Name()+"|"+path)
			content := mustReadFile(t, path)
			assertLineContainsFragments(t, content, "$env:GRAPES_OUTPUT_PATH = ")
			assertLineContainsFragments(t, content, "$env:GRAPES_EXEC_PATH = ", "zoxide")
			assertLineContainsFragments(t, content, "init powershell", "Set-Content", "zoxide.ps1")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	outputDir := expectedRunOutputDir(t, home, appData)
	assertFileExists(t, filepath.Join(outputDir, "cache"))
	assertFileExists(t, expectedRunStateDir(home))
	setupPath := filepath.Join(outputDir, "pwsh-setup.ps1")
	assertFileExists(t, setupPath)
	if got, want := len(executed), 1; got != want {
		t.Fatalf("len(executed) = %d, want %d", got, want)
	}
	if got, want := executed[0], "pwsh|"+setupPath; got != want {
		t.Fatalf("executed[0] = %q, want %q", got, want)
	}
	if !strings.Contains(stdout.String(), "Executed setup file "+setupPath) {
		t.Fatalf("stdout missing setup execution message: %q", stdout.String())
	}
	envContent := mustReadFile(t, filepath.Join(outputDir, "pwsh-env.ps1"))
	mainContent := mustReadFile(t, filepath.Join(outputDir, "pwsh-profile.ps1"))
	combined := envContent + "\n" + mainContent

	assertNoPosixBuiltInSyntax(t, combined)
	assertLineContainsFragments(t, envContent, "$env:GOPATH = ", "Join-Path", "go")
	assertLineContainsFragments(t, envContent, "$env:PATH = ", "Join-Path", "GOPATH")
	assertLineContainsFragments(t, envContent, "$env:BUN_INSTALL = ", "Join-Path", ".bun")
	assertLineContainsFragments(t, envContent, "$env:PATH = ", "Join-Path", "BUN_INSTALL")
	assertLineContainsFragments(t, envContent, "$env:GRAPES_OUT_CACHE_DIR = ", "Join-Path", "cache")
	assertLineContainsFragments(t, envContent, "$env:PATH = ", "GRAPES_EXEC_DIR")
	assertLineContainsFragments(t, envContent, "$env:GRAPES_EXEC_PATH = ", "fnm")
	assertLineContainsFragments(t, envContent, "$env:GRAPES_EXEC_VERSION = ", "1.39.0")
	assertLineContainsFragments(t, envContent, "fnm env --shell powershell", "Invoke-Expression")
	assertFileExcludes(t, combined, "fzf --bash")
	assertFileExcludes(t, combined, "fzf --zsh")
	assertLineContainsFragments(t, mainContent, "$env:GRAPES_EXEC_PATH = ", "zoxide")
	assertFileExcludes(t, mainContent, "FNM_PATH")
	assertLineContainsFragments(t, mainContent, "generate-shell-completion powershell")
	assertLineContainsFragments(t, mainContent, ". ~/.local/state/grapes/zoxide.ps1")
	assertFileExcludes(t, mainContent, "zoxide init powershell")
}

func TestRunDependencyChecksExecutableDependencyRendersWhenBinaryExists(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	createExecutable(t, binDir, "fnm", "echo 1.39.0")
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - fnm
---
`)
	writeTempFile(t, sourceDir, "fnm.grape", `---
phase: main
depend_executable:
  binary: fnm
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo fnm-fragment
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
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "fnm-fragment")
}

func TestRunEmitsScopedExecEnvironmentPerGrapeAndCleansUpAtFileEnd(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	execPath := createExecutable(t, binDir, "fnm", "echo 1.39.0")
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - fnm
  - prompt
---
`)
	writeTempFile(t, sourceDir, "fnm.grape", `---
phase: main
depend_executable:
  binary: fnm
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo fnm-fragment
`)
	writeTempFile(t, sourceDir, "prompt.grape", `---
phase: main
---
echo prompt-fragment
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
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), `export GRAPES_EXEC_PATH="`+strings.ReplaceAll(execPath, `\`, `/`)+`"`)
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), `export GRAPES_EXEC_DIR="`+strings.ReplaceAll(filepath.Dir(execPath), `\`, `/`)+`"`)
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), `export GRAPES_EXEC_VERSION="1.39.0"`)
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "# ==== grape: fnm")
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "# ==== grape: prompt")
	assertFileContains(t, filepath.Join(outputDir, "bashrc"), "# ==== cleanup variables")
	if got, want := strings.Count(content, "unset GRAPES_EXEC_PATH GRAPES_EXEC_DIR GRAPES_EXEC_VERSION"), 2; got != want {
		t.Fatalf("cleanup count = %d, want %d; content=%q", got, want, content)
	}
	if !strings.HasSuffix(content, "unset GRAPES_EXEC_PATH GRAPES_EXEC_DIR GRAPES_EXEC_VERSION\n") {
		t.Fatalf("bashrc did not end with scoped env cleanup: %q", content)
	}
}

func TestRunDependencyChecksExecutableDependencySkipsWhenBinaryMissing(t *testing.T) {
	home := t.TempDir()
	appData := ""
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	if runtime.GOOS == "windows" {
		appData = t.TempDir()
		t.Setenv("APPDATA", appData)
	}

	masterPath := writeTempFile(t, sourceDir, "master.grapes", `---
imports:
  - fnm
---
`)
	writeTempFile(t, sourceDir, "fnm.grape", `---
phase: main
depend_executable:
  binary: fnm
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
echo fnm-fragment
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
	assertFileExcludes(t, content, "fnm-fragment")
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

	assertFileMissing(t, filepath.Join(home, ".bashenv"))
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashenv"))+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashrc"))+`"`)
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
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashrc"))+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashenv"))+`"`)
	text := stdout.String()
	for _, fragment := range []string{
		"Linked files:",
		"linked " + filepath.Join(home, ".bashrc"),
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("stdout = %q, want fragment %q", text, fragment)
		}
	}
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
	var stdout bytes.Buffer
	if err := runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     []shells.Shell{target},
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       strings.NewReader("y\nn\n"),
		stdout:      &stdout,
		interactive: true,
	}); err != nil {
		t.Fatal(err)
	}

	assertFileMissing(t, filepath.Join(home, ".bashrc"))
	text := stdout.String()
	for _, fragment := range []string{
		"Linked files:",
		"skipped " + filepath.Join(home, ".bashrc"),
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("stdout = %q, want fragment %q", text, fragment)
		}
	}
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
	if err := shells.Install(filepath.Join(home, ".bashrc"), []string{
		`source "` + filepath.ToSlash(filepath.Join(outputDir, "bashenv")) + `"`,
		`source "` + filepath.ToSlash(filepath.Join(outputDir, "bashrc")) + `"`,
	}); err != nil {
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
	for _, fragment := range []string{
		"Linked files:",
		"unchanged " + filepath.Join(home, ".bashrc"),
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
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
	assertFileMissing(t, filepath.Join(home, ".bashenv"))
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashenv"))+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.ToSlash(filepath.Join(outputDir, "bashrc"))+`"`)
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

func expectedRunStateDir(home string) string {
	return filepath.Join(home, ".local", "state", "grapes")
}

func expectedInjectedOutputPath(shellName string, outputDir string) string {
	switch shellName {
	case "bash", "zsh":
		return strings.ReplaceAll(outputDir, `\`, "/")
	default:
		return outputDir
	}
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

func copyExampleFragments(t *testing.T, dir string, names ...string) {
	t.Helper()

	for _, name := range names {
		src := filepath.Join("..", "..", "docs", "grapes", name+".grape")
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read example fragment %s: %v", src, err)
		}
		writeTempFile(t, dir, name+".grape", string(data))
	}
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
