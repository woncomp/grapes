package parser

import (
	"fmt"
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

type frontmatter struct {
	Deps    []string          `yaml:"deps"`
	Phase   string            `yaml:"phase"`
	Env     map[string]string `yaml:"env"`
	Paths   []string          `yaml:"paths"`
	Imports []string          `yaml:"imports"`
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

func parseBlock(rb rawBlock, index int, path string) (Block, frontmatter, error) {
	parsed, err := parseFrontmatter(rb.Frontmatter)
	if err != nil {
		return Block{}, frontmatter{}, fmt.Errorf("parsing frontmatter block %d in %s: %w", index+1, path, err)
	}

	phase := parsed.Phase
	if phase == "" {
		phase = "main"
	}
	if phase != "env" && phase != "main" {
		return Block{}, frontmatter{}, fmt.Errorf("invalid phase %q in block %d of %s (must be \"env\" or \"main\")", phase, index+1, path)
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
