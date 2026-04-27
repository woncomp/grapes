package parser

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DependExecutable struct {
	Binary       string
	SearchPaths  []string
	VersionArgs  []string
	VersionRegex string
}

type GrapeFile struct {
	Name             string
	Path             string
	DependExecutable *DependExecutable
	Blocks           []Block
}

func ParseGrapeFile(path string) (*GrapeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return ParseGrapeString(name, string(data), path)
}

func ParseGrapeString(name, content, path string) (*GrapeFile, error) {
	grape := &GrapeFile{Name: name, Path: path}

	rawBlocks, err := splitBlocks(content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	for i, rb := range rawBlocks {
		block, parsed, err := parseBlock(rb, i, path)
		if err != nil {
			return nil, err
		}
		if len(parsed.Deps) > 0 {
			return nil, fmt.Errorf("deps is not supported in %s", path)
		}
		if i == 0 && parsed.DependExecutable != nil {
			grape.DependExecutable = &DependExecutable{
				Binary:       parsed.DependExecutable.Binary,
				SearchPaths:  append([]string(nil), parsed.DependExecutable.SearchPaths...),
				VersionArgs:  append([]string(nil), parsed.DependExecutable.VersionArgs...),
				VersionRegex: parsed.DependExecutable.VersionRegex,
			}
		}
		grape.Blocks = append(grape.Blocks, block)
	}

	if len(grape.Blocks) == 0 {
		grape.Blocks = append(grape.Blocks, Block{Phase: "main", Body: content})
	}

	return grape, nil
}

func ParseEmbeddedGrape(dir, name string, embedFS embed.FS) (*GrapeFile, error) {
	localPath := filepath.Join(dir, name+".grape")
	data, err := os.ReadFile(localPath)
	if err == nil {
		return ParseGrapeString(name, string(data), localPath)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading %s: %w", localPath, err)
	}

	embedData, embedErr := embedFS.ReadFile(name + ".grape")
	if embedErr != nil {
		return nil, fmt.Errorf("reading %s: %w", localPath, os.ErrNotExist)
	}
	return ParseGrapeString(name, string(embedData), "<embedded:"+name+">")
}
