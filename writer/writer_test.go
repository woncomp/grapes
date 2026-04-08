package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBasic(t *testing.T) {
	dir := t.TempDir()

	bashFragments := []Fragment{
		{Name: "path", Content: "export PATH=/bin\n"},
		{Name: "prompt", Content: "PS1='$ '\n"},
	}

	outputs := []ShellOutput{
		{Shell: "bash", Phase: "env", Fragments: bashFragments[:1]},
		{Shell: "bash", Phase: "main", Fragments: bashFragments[1:]},
		{Shell: "zsh", Phase: "env", Fragments: bashFragments[:1]},
		{Shell: "zsh", Phase: "main", Fragments: bashFragments[1:]},
	}

	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}

	// Check bashenv
	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("bashenv missing path content: %q", string(data))
	}

	// Check bashrc
	data, err = os.ReadFile(filepath.Join(dir, "bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PS1=") {
		t.Errorf("bashrc missing prompt content: %q", string(data))
	}

	// Check zshenv
	data, err = os.ReadFile(filepath.Join(dir, "zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("zshenv missing path content: %q", string(data))
	}

	// Check zshrc
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

	outputs := []ShellOutput{
		{Shell: "bash", Phase: "main", Fragments: nil},
	}

	if err := Write(target, outputs); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("expected output directory to be created")
	}
}

func TestWriteEmptyPhase(t *testing.T) {
	dir := t.TempDir()

	outputs := []ShellOutput{
		{Shell: "bash", Phase: "env", Fragments: nil},
		{Shell: "bash", Phase: "main", Fragments: []Fragment{{Name: "test", Content: "echo hi\n"}}},
	}

	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}

	// bashenv should exist but be empty
	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("bashenv should be empty, got %q", string(data))
	}
}
