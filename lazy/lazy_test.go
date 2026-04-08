package lazy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallFresh(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcFile, []byte("# existing content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Install(rcFile, "$HOME/.config/grapes/bashrc")
	if err != nil {
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

	// First install
	if err := os.WriteFile(rcFile, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(rcFile, "$HOME/.config/grapes/bashrc"); err != nil {
		t.Fatal(err)
	}

	// Second install with different path
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

	err := Install(rcFile, "$HOME/.config/grapes/bashrc")
	if err != nil {
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
	original := "# existing content\n"
	if err := os.WriteFile(rcFile, []byte(original), 0o644); err != nil {
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

func TestDetectBashProfile(t *testing.T) {
	dir := t.TempDir()

	// No files exist — should default to .bashenv
	target := DetectBashEnvTarget(dir)
	if !strings.HasSuffix(target, ".bashenv") {
		t.Errorf("expected .bashenv fallback, got %s", target)
	}

	// Create .bash_profile — should use that
	profile := filepath.Join(dir, ".bash_profile")
	if err := os.WriteFile(profile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	target = DetectBashEnvTarget(dir)
	if !strings.HasSuffix(target, ".bash_profile") {
		t.Errorf("expected .bash_profile, got %s", target)
	}
}
