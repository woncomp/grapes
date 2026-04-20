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

	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bashenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=/bin") {
		t.Errorf("bashenv missing path content: %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dir, "bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PS1=") {
		t.Errorf("bashrc missing prompt content: %q", string(data))
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

	if err := Write(target, outputs); err != nil {
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

	if err := Write(dir, outputs); err != nil {
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

func TestWriteMkdirAllError(t *testing.T) {
	outputs := []OutputFile{
		{Filename: "bashrc", Fragments: []Fragment{{Name: "test", Content: "hi\n"}}},
	}
	err := Write("/dev/null/subdir", outputs)
	if err == nil {
		t.Error("expected error for MkdirAll on /dev/null/subdir")
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
	err := Write(filepath.Join(blockFile, "sub"), outputs)
	if err == nil {
		t.Error("expected error when writing to blocked path")
	}
}
