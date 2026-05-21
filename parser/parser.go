package parser

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Block represents one frontmatter+body section within a .grape file.
type Block struct {
	Phase string            // "env", "main", or "setup"
	Env   map[string]string // first-class env vars
	Paths []string          // first-class PATH entries
	Body  string            // raw block content
}

type executableDependency struct {
	Binary       string   `yaml:"binary"`
	SearchPaths  []string `yaml:"search_paths"`
	VersionArgs  []string `yaml:"version_args"`
	VersionRegex string   `yaml:"version_regex"`
}

type fileDependency struct {
	Paths []string `yaml:"paths"`
}

type frontmatter struct {
	Deps             []string              `yaml:"deps"`
	Phase            string                `yaml:"phase"`
	Env              map[string]string     `yaml:"env"`
	Paths            []string              `yaml:"paths"`
	Imports          []string              `yaml:"imports"`
	DependExecutable *executableDependency `yaml:"depend_executable"`
	DependFile       *fileDependency       `yaml:"depend_file"`
}

type rawBlock struct {
	Frontmatter string
	Body        string
}

func parseFrontmatter(content string) (frontmatter, error) {
	var parsed frontmatter
	if strings.TrimSpace(content) == "" {
		return parsed, nil
	}
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		return frontmatter{}, err
	}
	return parsed, nil
}

func validateExecutableDependency(dep *executableDependency, path string) error {
	if dep == nil {
		return nil
	}
	if strings.TrimSpace(dep.Binary) == "" {
		return fmt.Errorf("invalid depend_executable in %s: binary is required", path)
	}
	if dep.VersionRegex != "" && len(dep.VersionArgs) == 0 {
		return fmt.Errorf("invalid depend_executable in %s: version_args is required when version_regex is set", path)
	}
	if dep.VersionRegex != "" {
		if _, err := regexp.Compile(dep.VersionRegex); err != nil {
			return fmt.Errorf("invalid depend_executable in %s: version_regex: %w", path, err)
		}
	}
	return nil
}

func validateFileDependency(dep *fileDependency, path string) error {
	if dep == nil {
		return nil
	}
	if len(dep.Paths) == 0 {
		return fmt.Errorf("invalid depend_file in %s: paths is required", path)
	}
	return nil
}

func parseBlock(rb rawBlock, index int, path string) (Block, frontmatter, error) {
	parsed, err := parseFrontmatter(rb.Frontmatter)
	if err != nil {
		return Block{}, frontmatter{}, fmt.Errorf("parsing frontmatter block %d in %s: %w", index+1, path, err)
	}

	if err := validateExecutableDependency(parsed.DependExecutable, path); err != nil {
		return Block{}, frontmatter{}, err
	}
	if err := validateFileDependency(parsed.DependFile, path); err != nil {
		return Block{}, frontmatter{}, err
	}

	phase := parsed.Phase
	if phase == "" {
		phase = "main"
	}
	if phase != "env" && phase != "main" && phase != "setup" {
		return Block{}, frontmatter{}, fmt.Errorf("invalid phase %q in block %d of %s (must be \"env\", \"main\", or \"setup\")", phase, index+1, path)
	}

	return Block{
		Phase: phase,
		Env:   parsed.Env,
		Paths: parsed.Paths,
		Body:  rb.Body,
	}, parsed, nil
}

func splitBlocks(content string) ([]rawBlock, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return []rawBlock{{Body: content}}, nil
	}

	var blocks []rawBlock
	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) != "---" {
			break
		}
		i++

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
		i++

		bodyStart := i
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == "---" {
				break
			}
			i++
		}

		blocks = append(blocks, rawBlock{
			Frontmatter: fmContent,
			Body:        strings.Join(lines[bodyStart:i], "\n"),
		})
	}

	return blocks, nil
}
