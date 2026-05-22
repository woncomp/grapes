package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBasic(t *testing.T) {
	dir := t.TempDir()

	fragments := []Fragment{
		{Name: "path", Content: "export PATH=/bin\n"},
		{Name: "prompt", Content: "PS1='$ '\n"},
	}

	outputs := []OutputFile{
		{Filename: "bashenv", Fragments: fragments[:1]},
		{Filename: "bashrc", Fragments: fragments[1:]},
		{Filename: "zshenv", Fragments: fragments[:1]},
		{Filename: "zshrc", Fragments: fragments[1:]},
	}

	if err := Write("linux", dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("bashenv missing path content: %q", string(data))
	}
	if !strings.Contains(string(data), "# ==== grape: path") {
		t.Errorf("bashenv missing fragment divider: %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dir, "bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PS1=") {
		t.Errorf("bashrc missing prompt content: %q", string(data))
	}
	if !strings.Contains(string(data), "# ==== grape: prompt") {
		t.Errorf("bashrc missing fragment divider: %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dir, "zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("zshenv missing path content: %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dir, "zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PS1=") {
		t.Errorf("zshrc missing prompt content: %q", string(data))
	}
}

func TestWriteCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "subdir", "grapes")

	outputs := []OutputFile{
		{Filename: "bashrc"},
	}

	if err := Write("linux", target, outputs); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("expected output directory to be created")
	}
}

func TestWriteEmptyPhase(t *testing.T) {
	dir := t.TempDir()

	outputs := []OutputFile{
		{Filename: "bashenv"},
		{Filename: "bashrc", Fragments: []Fragment{{Name: "test", Content: "echo hi\n"}}},
	}

	if err := Write("linux", dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("bashenv should be empty, got %q", string(data))
	}
}

func TestWriteSkipsDividerForInternalFragments(t *testing.T) {
	dir := t.TempDir()

	outputs := []OutputFile{
		{
			Filename: "bashenv",
			Fragments: []Fragment{
				{Name: "__GRAPE_ENV", Content: "export GRAPES=1\n"},
				{Name: "fnm", Content: "eval \"$(fnm env --use-on-cd)\"\n"},
			},
		},
	}

	if err := Write("linux", dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.HasPrefix(got, "export GRAPES=1\n\n# =============================================\n# ==== grape: fnm\n\n") {
		t.Fatalf("bashenv formatting mismatch: %q", got)
	}
	if strings.Contains(got, "# ==== grape: __GRAPE_ENV") {
		t.Fatalf("internal fragment unexpectedly rendered divider: %q", got)
	}
}

func TestWriteAddsCleanupDividerForCleanupFragment(t *testing.T) {
	dir := t.TempDir()

	outputs := []OutputFile{
		{
			Filename: "bashrc",
			Fragments: []Fragment{
				{Name: "fnm", Content: "echo fnm\n"},
				{Name: "__GRAPE_SCOPE_CLEANUP", Content: "unset GRAPES_EXEC_PATH GRAPES_EXEC_DIR\n"},
			},
		},
	}

	if err := Write("linux", dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bashrc"))
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "# ==== cleanup variables") {
		t.Fatalf("cleanup divider missing: %q", got)
	}
	if !strings.Contains(got, "unset GRAPES_EXEC_PATH GRAPES_EXEC_DIR\n") {
		t.Fatalf("cleanup content missing: %q", got)
	}
	if strings.Contains(got, "# ==== grape: __GRAPE_SCOPE_CLEANUP") {
		t.Fatalf("cleanup fragment used grape divider: %q", got)
	}
}

func TestWriteMkdirAllError(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	outputs := []OutputFile{
		{Filename: "bashrc", Fragments: []Fragment{{Name: "test", Content: "hi\n"}}},
	}
	err := Write("linux", filepath.Join(blocked, "subdir"), outputs)
	if err == nil {
		t.Error("expected error for MkdirAll on blocked path")
	}
}

func TestWriteFileError(t *testing.T) {
	dir := t.TempDir()
	blockFile := filepath.Join(dir, "output")
	if err := os.WriteFile(blockFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	outputs := []OutputFile{
		{Filename: "bashrc", Fragments: []Fragment{{Name: "test", Content: "hi\n"}}},
	}
	err := Write("linux", filepath.Join(blockFile, "sub"), outputs)
	if err == nil {
		t.Error("expected error when writing to blocked path")
	}
}

func TestWriteNormalizesLinuxLineEndings(t *testing.T) {
	dir := t.TempDir()

	outputs := []OutputFile{
		{
			Filename: "bashrc",
			Fragments: []Fragment{
				{Name: "__GRAPE_ENV", Content: "export GRAPES=1\r\n"},
				{Name: "fnm", Content: "echo fnm\r\necho node\r\n"},
			},
		},
	}

	if err := Write("linux", dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bashrc"))
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if strings.Contains(got, "\r") {
		t.Fatalf("linux output should use LF only, got %q", got)
	}
}

func TestWritePreservesWindowsLineEndings(t *testing.T) {
	dir := t.TempDir()

	outputs := []OutputFile{
		{
			Filename: "pwsh-profile.ps1",
			Fragments: []Fragment{
				{Name: "fnm", Content: "Write-Host 'fnm'\r\n"},
			},
		},
	}

	if err := Write("windows", dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "pwsh-profile.ps1"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "\r\n") {
		t.Fatalf("windows output should preserve CRLF content, got %q", string(data))
	}
}
