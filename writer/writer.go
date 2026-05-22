package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
func Write(goos string, targetDir string, outputs []OutputFile) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	for _, out := range outputs {
		path := filepath.Join(targetDir, out.Filename)

		var content string
		for _, f := range out.Fragments {
			content += renderFragment(f)
		}
		content = normalizeLineEndings(goos, content)

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

func renderFragment(fragment Fragment) string {
	name := strings.TrimSpace(fragment.Name)
	switch {
	case name == "__GRAPE_SCOPE_CLEANUP":
		return fmt.Sprintf("\n# =============================================\n# ==== cleanup variables\n\n%s", fragment.Content)
	case name == "" || strings.HasPrefix(name, "__"):
		return fragment.Content
	}

	return fmt.Sprintf("\n# =============================================\n# ==== grape: %s\n\n%s", name, fragment.Content)
}

func normalizeLineEndings(goos string, content string) string {
	if goos == "windows" {
		return content
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}
