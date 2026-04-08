package lazy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	markerStart = "# >>> grapes >>>"
	markerEnd   = "# <<< grapes <<<"
)

// Install adds or updates a marker block in rcFile that sources sourcePath.
// If a marker block already exists, it is replaced.
func Install(rcFile string, sourcePath string) error {
	sourceLine := fmt.Sprintf("source \"%s\"", sourcePath)
	block := markerStart + "\n" + sourceLine + "\n" + markerEnd + "\n"

	var existing string
	if data, err := os.ReadFile(rcFile); err == nil {
		existing = string(data)
	}

	// Remove existing marker block if present
	if strings.Contains(existing, markerStart) {
		existing = removeMarkerBlock(existing)
	}

	// Append the new block
	content := strings.TrimRight(existing, "\n") + "\n" + block

	return os.WriteFile(rcFile, []byte(content), 0o644)
}

// Uninstall removes the marker block from rcFile, preserving other content.
func Uninstall(rcFile string) error {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := removeMarkerBlock(string(data))
	return os.WriteFile(rcFile, []byte(content), 0o644)
}

// removeMarkerBlock removes everything between and including marker delimiters.
func removeMarkerBlock(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inBlock := false

	for _, line := range lines {
		if strings.TrimSpace(line) == markerStart {
			inBlock = true
			continue
		}
		if strings.TrimSpace(line) == markerEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			result = append(result, line)
		}
	}

	joined := strings.Join(result, "\n")
	// Clean up any resulting blank lines
	for strings.Contains(joined, "\n\n\n") {
		joined = strings.ReplaceAll(joined, "\n\n\n", "\n\n")
	}
	return joined
}

// DetectBashEnvTarget returns the path to use for the bash env source file.
// Prefers ~/.bash_profile if it exists, otherwise ~/.bashenv.
func DetectBashEnvTarget(homeDir string) string {
	profile := filepath.Join(homeDir, ".bash_profile")
	if _, err := os.Stat(profile); err == nil {
		return profile
	}
	return filepath.Join(homeDir, ".bashenv")
}
