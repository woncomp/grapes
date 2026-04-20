package shells

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportedNames(t *testing.T) {
	names := SupportedNames()
	if got, want := strings.Join(names, ","), "bash,zsh"; got != want {
		t.Fatalf("SupportedNames() = %q, want %q", got, want)
	}
}

func TestParse(t *testing.T) {
	shell, err := Parse("ZSH")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.Name(), "zsh"; got != want {
		t.Fatalf("Parse(ZSH).Name() = %q, want %q", got, want)
	}
}

func TestParseUnsupported(t *testing.T) {
	_, err := Parse("fish")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unsupported target "fish"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectCurrent(t *testing.T) {
	shell, err := DetectCurrent(func(key string) (string, bool) {
		if key == "SHELL" {
			return "/bin/zsh", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.Name(), "zsh"; got != want {
		t.Fatalf("DetectCurrent().Name() = %q, want %q", got, want)
	}
}

func TestDetectCurrentFailsWithoutShell(t *testing.T) {
	_, err := DetectCurrent(func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not detect current shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallFresh(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte("# existing content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "# existing content") {
		t.Error("existing content was lost")
	}
	if !strings.Contains(content, "source \"$HOME/.config/grapes/bashrc\"") {
		t.Error("missing source line")
	}
	if !strings.Contains(content, markerStart) {
		t.Error("missing marker start")
	}
	if !strings.Contains(content, markerEnd) {
		t.Error("missing marker end")
	}
}

func TestInstallUpdate(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")

	if err := os.WriteFile(rcFile, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}
	if err := Install(rcFile, "$HOME/.config/grapes/new-bashrc"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Count(content, markerStart) != 1 {
		t.Errorf("should have exactly one marker block, found %d", strings.Count(content, markerStart))
	}
	if !strings.Contains(content, "new-bashrc") {
		t.Error("missing updated source line")
	}
}

func TestInstallEmptyFile(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "source") {
		t.Error("missing source line in empty file")
	}
}

func TestUninstall(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte("# existing content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(rcFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, markerStart) {
		t.Error("marker should be removed after uninstall")
	}
	if strings.Contains(content, "source") {
		t.Error("source line should be removed after uninstall")
	}
	if !strings.Contains(content, "# existing content") {
		t.Error("existing content should be preserved after uninstall")
	}
}

func TestUninstallNonExistent(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := Uninstall(rcFile); err != nil {
		t.Fatal(err)
	}
}
