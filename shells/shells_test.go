package shells

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportedNames(t *testing.T) {
	names := SupportedNames()
	if got, want := strings.Join(names, ","), "pwsh,nushell,zsh,bash"; got != want {
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

func TestParseDoesNotSupportLegacyWindowsPSTargetName(t *testing.T) {
	_, err := Parse("powershell")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unsupported target "powershell"`) {
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

func TestDetectCurrentUsesPwshProcessAncestor(t *testing.T) {
	shell, err := detectCurrent(func(string) (string, bool) {
		return "", false
	}, func() []string {
		return []string{"pwsh.exe"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.Name(), "pwsh"; got != want {
		t.Fatalf("DetectCurrent().Name() = %q, want %q", got, want)
	}
}

func TestDetectCurrentUsesSupportedProcessAncestorThroughGoRun(t *testing.T) {
	shell, err := detectCurrent(func(string) (string, bool) {
		return "", false
	}, func() []string {
		return []string{"go.exe", "go.exe", "pwsh.exe"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.Name(), "pwsh"; got != want {
		t.Fatalf("DetectCurrent().Name() = %q, want %q", got, want)
	}
}

func TestDetectCurrentUsesSupportedProcessAncestorThroughWindowsGoRun(t *testing.T) {
	shell, err := detectCurrent(func(string) (string, bool) {
		return "", false
	}, func() []string {
		return []string{"cmd.exe", "go.exe", "pwsh.exe"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shell.Name(), "pwsh"; got != want {
		t.Fatalf("DetectCurrent().Name() = %q, want %q", got, want)
	}
}

func TestDetectCurrentDoesNotUseShellBehindUnsupportedParent(t *testing.T) {
	_, err := detectCurrent(func(string) (string, bool) {
		return "", false
	}, func() []string {
		return []string{"cmd.exe", "pwsh.exe"}
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not detect current shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectCurrentFailsWithoutShell(t *testing.T) {
	_, err := detectCurrent(func(string) (string, bool) {
		return "", false
	}, func() []string {
		return nil
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

	if err := Install(rcFile, []string{`source "$HOME/.config/grapes/bashrc"`}); err != nil {
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
	if err := Install(rcFile, []string{`source "$HOME/.config/grapes/bashrc"`}); err != nil {
		t.Fatal(err)
	}
	if err := Install(rcFile, []string{`source "$HOME/.config/grapes/new-bashrc"`}); err != nil {
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

	if err := Install(rcFile, []string{`source "$HOME/.config/grapes/bashrc"`}); err != nil {
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

	if err := Install(rcFile, []string{`source "$HOME/.config/grapes/bashrc"`}); err != nil {
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

func TestInstallSupportsMultipleLines(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, "PowerShell", "Microsoft.PowerShell_profile.ps1")

	err := Install(rcFile, []string{
		`. "$HOME/.config/grapes/pwsh-env.ps1"`,
		`. "$HOME/.config/grapes/pwsh-profile.ps1"`,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, `. "$HOME/.config/grapes/pwsh-env.ps1"`) {
		t.Fatalf("missing env install line: %q", content)
	}
	if !strings.Contains(content, `. "$HOME/.config/grapes/pwsh-profile.ps1"`) {
		t.Fatalf("missing main install line: %q", content)
	}
}

func TestInstallCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, "nested", "profile")

	if err := Install(rcFile, []string{`source "$HOME/.config/grapes/bashrc"`}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Dir(rcFile)); err != nil {
		t.Fatalf("expected parent directory to exist: %v", err)
	}
}

func TestInstallReturnsContextOnParentDirectoryCreationFailure(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "blocked")
	if err := os.WriteFile(parent, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	rcFile := filepath.Join(parent, "profile")
	err := Install(rcFile, []string{`source "$HOME/.config/grapes/bashrc"`})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "creating rc directory " + filepath.Dir(rcFile)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Install() error = %q, want substring %q", err.Error(), want)
	}
}
