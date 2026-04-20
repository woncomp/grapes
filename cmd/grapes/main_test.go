package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	_, err := parseArgs([]string{"master.grapes"}, func(string) (string, bool) {
		return "", false
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
	if !strings.Contains(usage, "Usage: grapes <input> [-t shell]... [--nolink]") {
		t.Fatalf("usage did not contain new command shape: %s", usage)
	}
	if strings.Contains(usage, "--lazy") {
		t.Fatalf("usage should not mention --lazy: %s", usage)
	}
}

func TestRunNoLinkGeneratesOnlySelectedTargets(t *testing.T) {
	home := t.TempDir()
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)

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

	if err := run(masterPath, []shells.Shell{target}, true); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(home, ".config", "grapes")
	assertFileExists(t, filepath.Join(outputDir, "zshenv"))
	assertFileExists(t, filepath.Join(outputDir, "zshrc"))
	assertFileMissing(t, filepath.Join(outputDir, "bashenv"))
	assertFileMissing(t, filepath.Join(outputDir, "bashrc"))
	assertFileMissing(t, filepath.Join(home, ".zshenv"))
	assertFileMissing(t, filepath.Join(home, ".zshrc"))
}

func TestRunLinksOnlySelectedTarget(t *testing.T) {
	home := t.TempDir()
	sourceDir := t.TempDir()
	t.Setenv("HOME", home)

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

	if err := run(masterPath, []shells.Shell{target}, false); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(home, ".config", "grapes")
	assertFileExists(t, filepath.Join(outputDir, "bashenv"))
	assertFileExists(t, filepath.Join(outputDir, "bashrc"))
	assertFileMissing(t, filepath.Join(home, ".zshenv"))
	assertFileMissing(t, filepath.Join(home, ".zshrc"))

	assertFileContains(t, filepath.Join(home, ".bashenv"), `source "`+filepath.Join(outputDir, "bashenv")+`"`)
	assertFileContains(t, filepath.Join(home, ".bashrc"), `source "`+filepath.Join(outputDir, "bashrc")+`"`)
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
