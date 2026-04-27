package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GrapesFile struct {
	Name    string
	Path    string
	Imports []string
}

func ParseGrapesFile(path string) (*GrapesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return parseGrapesContent(name, string(data), path)
}

func parseGrapesContent(name, content, path string) (*GrapesFile, error) {
	grapes := &GrapesFile{Name: name, Path: path}

	rawBlocks, err := splitBlocks(content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(rawBlocks) == 0 {
		return grapes, nil
	}

	parsed, err := parseFrontmatter(rawBlocks[0].Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter block 1 in %s: %w", path, err)
	}
	grapes.Imports = parsed.Imports
	return grapes, nil
}
