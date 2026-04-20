package writer

import (
	"fmt"
	"os"
	"path/filepath"
)

// Fragment is a preprocessed fragment ready for output.
type Fragment struct {
	Name    string
	Content string
}

// OutputFile represents one managed file to write.
type OutputFile struct {
	Filename  string
	Fragments []Fragment
}

// Write generates output files in the target directory.
// Creates the directory if it doesn't exist.
func Write(targetDir string, outputs []OutputFile) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	for _, out := range outputs {
		path := filepath.Join(targetDir, out.Filename)

		var content string
		for _, f := range out.Fragments {
			content += f.Content
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}
