package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type GrapeImport struct {
	From         string
	Import       string
	ResolvedPath string
	Key          string
	Label        string
}

type GrapesFile struct {
	Name    string
	Path    string
	Imports []GrapeImport
}

type grapesDocument struct {
	Grapes []grapesImport `toml:"grape"`
}

type grapesImport struct {
	From   string `toml:"from"`
	Import string `toml:"import"`
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

	normalizedPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("normalizing %s: %w", path, err)
	}

	var doc grapesDocument
	if err := toml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("parsing TOML in %s: %w", path, err)
	}

	baseDir := filepath.Dir(normalizedPath)
	for i, entry := range doc.Grapes {
		normalized, err := normalizeGrapeImport(baseDir, entry)
		if err != nil {
			return nil, fmt.Errorf("parsing grape entry %d in %s: %w", i+1, path, err)
		}
		grapes.Imports = append(grapes.Imports, normalized)
	}

	return grapes, nil
}

func normalizeGrapeImport(baseDir string, entry grapesImport) (GrapeImport, error) {
	rawImport := strings.TrimSpace(entry.Import)
	if rawImport == "" {
		return GrapeImport{}, fmt.Errorf("import is required")
	}

	importPath := rawImport
	if filepath.Ext(importPath) == "" {
		importPath += ".grape"
	}
	if filepath.Ext(importPath) != ".grape" {
		return GrapeImport{}, fmt.Errorf("import %q must target a .grape file", entry.Import)
	}

	sourceDir := baseDir
	if trimmedFrom := strings.TrimSpace(entry.From); trimmedFrom != "" {
		sourceDir = filepath.Clean(filepath.Join(baseDir, trimmedFrom))
	}

	resolvedPath, err := filepath.Abs(filepath.Clean(filepath.Join(sourceDir, importPath)))
	if err != nil {
		return GrapeImport{}, fmt.Errorf("resolving import %q: %w", entry.Import, err)
	}

	key, err := filepath.Rel(baseDir, resolvedPath)
	if err != nil {
		return GrapeImport{}, fmt.Errorf("deriving key for import %q: %w", entry.Import, err)
	}
	key = filepath.Clean(key)
	label := strings.TrimSuffix(filepath.ToSlash(key), ".grape")

	return GrapeImport{
		From:         strings.TrimSpace(entry.From),
		Import:       rawImport,
		ResolvedPath: resolvedPath,
		Key:          filepath.ToSlash(key),
		Label:        label,
	}, nil
}
