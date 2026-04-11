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

// ShellOutput represents all fragments for one shell+phase combination.
type ShellOutput struct {
	Shell     string // "bash" or "zsh"
	Phase     string // "env" or "main"
	Fragments []Fragment
}

// Write generates output files in the target directory.
// Creates the directory if it doesn't exist.
// Output files: {shell}{phase_suffix} where phase_suffix is "" for main, "env" for env.
func Write(targetDir string, outputs []ShellOutput) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	for _, out := range outputs {
		filename := out.Shell + phaseSuffix(out.Phase)
		path := filepath.Join(targetDir, filename)

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

func phaseSuffix(phase string) string {
	switch phase {
	case "env":
		return "env"
	default:
		return "rc"
	}
}
