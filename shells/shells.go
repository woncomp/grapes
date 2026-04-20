package shells

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	PhaseEnv  = "env"
	PhaseMain = "main"

	markerStart = "# >>> grapes >>>"
	markerEnd   = "# <<< grapes <<<"
)

type Shell interface {
	Name() string
	Aliases() []string
	ManagedFilename(phase string) string
	LinkTargets(home, outputDir string) []LinkTarget
}

type LinkTarget struct {
	RCFile     string
	SourcePath string
}

var (
	supportedShells []Shell
	shellByAlias    = map[string]Shell{}
)

func Supported() []Shell {
	return append([]Shell(nil), supportedShells...)
}

func SupportedNames() []string {
	names := make([]string, 0, len(supportedShells))
	for _, shell := range supportedShells {
		names = append(names, shell.Name())
	}
	slices.Sort(names)
	return names
}

func Parse(raw string) (Shell, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	shell, ok := shellByAlias[normalized]
	if ok {
		return shell, nil
	}

	return nil, fmt.Errorf("unsupported target %q (supported: %s)", raw, strings.Join(SupportedNames(), ", "))
}

func DetectCurrent(lookupEnv func(string) (string, bool)) (Shell, error) {
	shellPath, ok := lookupEnv("SHELL")
	if !ok || strings.TrimSpace(shellPath) == "" {
		return nil, fmt.Errorf("could not detect current shell; pass -t with one of: %s", strings.Join(SupportedNames(), ", "))
	}

	detected := strings.TrimPrefix(filepath.Base(shellPath), "-")
	shell, err := Parse(detected)
	if err != nil {
		return nil, fmt.Errorf("could not use detected shell %q; pass -t with one of: %s", detected, strings.Join(SupportedNames(), ", "))
	}

	return shell, nil
}

// Install adds or updates a marker block in rcFile that sources sourcePath.
// If a marker block already exists, it is replaced.
func Install(rcFile string, sourcePath string) error {
	sourceLine := fmt.Sprintf("source \"%s\"", sourcePath)
	block := markerStart + "\n" + sourceLine + "\n" + markerEnd + "\n"

	var existing string
	if data, err := os.ReadFile(rcFile); err == nil {
		existing = string(data)
	}

	if strings.Contains(existing, markerStart) {
		existing = removeMarkerBlock(existing)
	}

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

func registerShell(shell Shell) {
	supportedShells = append(supportedShells, shell)
	shellByAlias[shell.Name()] = shell
	for _, alias := range shell.Aliases() {
		shellByAlias[strings.ToLower(strings.TrimSpace(alias))] = shell
	}
}

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
	for strings.Contains(joined, "\n\n\n") {
		joined = strings.ReplaceAll(joined, "\n\n\n", "\n\n")
	}
	return joined
}
