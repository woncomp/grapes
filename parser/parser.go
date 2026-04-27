package parser

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Block represents one frontmatter+body section within a .grape file.
type Block struct {
	Phase string            // "env" or "main"
	Env   map[string]string // first-class env vars
	Paths []string          // first-class PATH entries
	Body  string            // raw block content
}

// Fragment represents a parsed .grape or .grapes file.
type Fragment struct {
	Name     string   // filename without extension
	Path     string   // full file path
	Deps     []string // fragment dependencies (from first block only)
	Imports  []string // master-only: fragments to include
	IsMaster bool     // true if this is a .grapes file
	Blocks   []Block  // multi-block structure (replaces old Phase + Body)
}

// frontmatter is the YAML structure parsed from between --- delimiters.
type frontmatter struct {
	Deps    []string          `yaml:"deps"`
	Phase   string            `yaml:"phase"`
	Env     map[string]string `yaml:"env"`
	Paths   []string          `yaml:"paths"`
	Imports []string          `yaml:"imports"`
}

// rawBlock holds the raw frontmatter YAML and body string before parsing.
type rawBlock struct {
	Frontmatter string // YAML content between --- delimiters (empty if no frontmatter)
	Body        string // body content after closing ---
}

// ParseFile reads a .grape or .grapes file and returns a Fragment.
func ParseFile(path string) (*Fragment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	isMaster := filepath.Ext(path) == ".grapes"
	frag, err := parseContent(name, string(data), path, isMaster)
	if err != nil {
		return nil, err
	}
	return frag, nil
}

// ParseString parses a fragment from string content (used for embedded fragments).
func ParseString(name, content string) (*Fragment, error) {
	return parseContent(name, content, "<embedded:"+name+">", false)
}

// ParseFileOrEmbedded tries to load a fragment from a local directory first,
// falling back to the embedded filesystem.
func ParseFileOrEmbedded(dir, name string, embedFS embed.FS) (*Fragment, error) {
	localPath := filepath.Join(dir, name+".grape")
	data, err := os.ReadFile(localPath)
	if err == nil {
		frag, err := parseContent(name, string(data), localPath, false)
		return frag, err
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading %s: %w", localPath, err)
	}

	// Fall back to embedded
	embedData, embedErr := embedFS.ReadFile(name + ".grape")
	if embedErr != nil {
		return nil, fmt.Errorf("reading %s: %w", localPath, os.ErrNotExist)
	}
	return ParseString(name, string(embedData))
}

// parseContent is the shared internal parser for any content source.
func parseContent(name, content, path string, isMaster bool) (*Fragment, error) {
	frag := &Fragment{
		Name:     name,
		Path:     path,
		IsMaster: isMaster,
	}

	rawBlocks, splitErr := splitBlocks(content)
	if splitErr != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, splitErr)
	}

	for i, rb := range rawBlocks {
		var parsed frontmatter
		if rb.Frontmatter != "" {
			if err := yaml.Unmarshal([]byte(rb.Frontmatter), &parsed); err != nil {
				return nil, fmt.Errorf("parsing frontmatter block %d in %s: %w", i+1, path, err)
			}
		}

		// deps only allowed in first block
		if i > 0 && len(parsed.Deps) > 0 {
			return nil, fmt.Errorf("deps not allowed in block %d of %s (only first block)", i+1, path)
		}

		// imports only in master files, first block
		if i > 0 && len(parsed.Imports) > 0 {
			return nil, fmt.Errorf("imports not allowed in block %d of %s (only first block)", i+1, path)
		}

		// First block captures deps/imports
		if i == 0 {
			frag.Deps = parsed.Deps
			frag.Imports = parsed.Imports
		}

		// Default phase
		phase := parsed.Phase
		if phase == "" {
			phase = "main"
		}

		// Validate phase
		if phase != "env" && phase != "main" {
			return nil, fmt.Errorf("invalid phase %q in block %d of %s (must be \"env\" or \"main\")", phase, i+1, path)
		}

		block := Block{
			Phase: phase,
			Env:   parsed.Env,
			Paths: parsed.Paths,
			Body:  rb.Body,
		}
		frag.Blocks = append(frag.Blocks, block)
	}

	// Ensure at least one block
	if len(frag.Blocks) == 0 {
		frag.Blocks = append(frag.Blocks, Block{
			Phase: "main",
			Body:  content,
		})
	}

	return frag, nil
}

// splitBlocks splits content into multiple rawBlock structures.
// Each frontmatter section is delimited by --- on its own line.
// Content without leading --- is treated as a single body-only block.
func splitBlocks(content string) ([]rawBlock, error) {
	lines := strings.Split(content, "\n")

	// If content doesn't start with ---, it's a body-only block
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return []rawBlock{{Body: content}}, nil
	}

	var blocks []rawBlock
	i := 0

	for i < len(lines) {
		// Find opening ---
		if strings.TrimSpace(lines[i]) != "---" {
			break
		}
		i++ // skip opening ---

		// Find closing ---
		fmStart := i
		end := -1
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
			i++
		}

		if end == -1 {
			return nil, fmt.Errorf("unterminated frontmatter (missing closing ---)")
		}

		fmContent := strings.Join(lines[fmStart:end], "\n")
		i++ // skip closing ---

		// Collect body until next --- or EOF
		bodyStart := i
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == "---" {
				break
			}
			i++
		}

		bodyContent := strings.Join(lines[bodyStart:i], "\n")
		blocks = append(blocks, rawBlock{
			Frontmatter: fmContent,
			Body:        bodyContent,
		})
	}

	return blocks, nil
}
