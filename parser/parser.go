package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Fragment represents a parsed .grape or .grapes file.
type Fragment struct {
	Name     string   // filename without extension
	Path     string   // full file path
	Phase    string   // "env" or "main"
	Deps     []string // fragment dependencies
	Imports  []string // master-only: fragments to include
	IsMaster bool     // true if this is a .grapes file
	Body     string   // raw body after frontmatter
}

// frontmatter is the YAML structure parsed from between --- delimiters.
type frontmatter struct {
	Deps    []string `yaml:"deps"`
	Phase   string   `yaml:"phase"`
	Imports []string `yaml:"imports"`
}

// ParseFile reads a .grape or .grapes file and returns a Fragment.
func ParseFile(path string) (*Fragment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	content := string(data)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	isMaster := filepath.Ext(path) == ".grapes"

	frag := &Fragment{
		Name:     name,
		Path:     path,
		Phase:    "main",
		IsMaster: isMaster,
	}

	// Split on --- delimiters for frontmatter
	body, fm, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if fm != nil {
		var parsed frontmatter
		if err := yaml.Unmarshal([]byte(*fm), &parsed); err != nil {
			return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
		}
		frag.Deps = parsed.Deps
		frag.Imports = parsed.Imports
		if parsed.Phase != "" {
			frag.Phase = parsed.Phase
		}
	}

	frag.Body = body

	// Validate phase
	if frag.Phase != "env" && frag.Phase != "main" {
		return nil, fmt.Errorf("invalid phase %q in %s (must be \"env\" or \"main\")", frag.Phase, path)
	}

	return frag, nil
}

// splitFrontmatter splits content into body and optional YAML frontmatter.
// Frontmatter is delimited by --- on its own line.
func splitFrontmatter(content string) (body string, fm *string, err error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content, nil, nil
	}

	// Find closing ---
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}

	if end == -1 {
		return "", nil, fmt.Errorf("unterminated frontmatter (missing closing ---)")
	}

	frontmatterContent := strings.Join(lines[1:end], "\n")
	bodyContent := strings.Join(lines[end+1:], "\n")

	return bodyContent, &frontmatterContent, nil
}
