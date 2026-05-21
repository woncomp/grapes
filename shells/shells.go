package shells

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PhaseEnv   = "env"
	PhaseMain  = "main"
	PhaseSetup = "setup"

	markerStart = "# >>> grapes >>>"
	markerEnd   = "# <<< grapes <<<"
)

type Shell interface {
	Name() string
	Aliases() []string
	ManagedFilename(phase string) string
	LinkTargets(TargetContext) ([]LinkTarget, error)
}

type TargetContext struct {
	GOOS      string
	LookupEnv func(string) (string, bool)
	OutputDir string
}

type LinkTarget struct {
	RCFile       string
	InstallLines []string
}

var (
	supportedShells []Shell
	shellByAlias    = map[string]Shell{}
)

func Supported() []Shell {
	return append([]Shell(nil), supportedShells...)
}

func SupportedNames() []string {
	return []string{"pwsh", "nushell", "zsh", "bash"}
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
	return detectCurrent(lookupEnv, processAncestorNames)
}

func detectCurrent(lookupEnv func(string) (string, bool), processAncestors func() []string) (Shell, error) {
	shellPath, ok := lookupEnv("SHELL")
	if ok && strings.TrimSpace(shellPath) != "" {
		detected := shellNameFromPath(shellPath)
		shell, err := Parse(detected)
		if err != nil {
			return nil, fmt.Errorf("could not use detected shell %q; pass -t with one of: %s", detected, strings.Join(SupportedNames(), ", "))
		}

		return shell, nil
	}

	if processAncestors != nil {
		if shell, ok := shellFromProcessAncestors(processAncestors()); ok {
			return shell, nil
		}
	}

	return nil, fmt.Errorf("could not detect current shell; pass -t with one of: %s", strings.Join(SupportedNames(), ", "))
}

func shellFromProcessAncestors(ancestors []string) (Shell, bool) {
	if len(ancestors) == 0 {
		return nil, false
	}

	parent := shellNameFromPath(ancestors[0])
	if shell, err := Parse(parent); err == nil {
		return shell, true
	}

	switch {
	case parent == "go":
		return shellAfterGoWrappers(ancestors[1:])
	case parent == "cmd" && len(ancestors) >= 2 && shellNameFromPath(ancestors[1]) == "go":
		return shellAfterGoWrappers(ancestors[2:])
	default:
		return nil, false
	}
}

func shellAfterGoWrappers(ancestors []string) (Shell, bool) {
	for len(ancestors) > 0 && shellNameFromPath(ancestors[0]) == "go" {
		ancestors = ancestors[1:]
	}
	if len(ancestors) == 0 {
		return nil, false
	}
	shell, err := Parse(shellNameFromPath(ancestors[0]))
	return shell, err == nil
}

func shellNameFromPath(raw string) string {
	name := strings.TrimSpace(raw)
	if index := strings.LastIndexAny(name, `\/`); index >= 0 {
		name = name[index+1:]
	}
	name = strings.TrimPrefix(name, "-")
	if strings.EqualFold(filepath.Ext(name), ".exe") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return name
}

func homeDir(goos string, lookupEnv func(string) (string, bool)) (string, error) {
	keys := []string{"HOME"}
	if goos == "windows" {
		keys = []string{"HOME", "USERPROFILE"}
	}
	for _, key := range keys {
		if value, ok := lookupEnv(key); ok && strings.TrimSpace(value) != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("home directory environment variable not set")
}

func configDir(goos string, lookupEnv func(string) (string, bool), appName string) (string, error) {
	if goos == "windows" {
		appData, ok := lookupEnv("APPDATA")
		if !ok || strings.TrimSpace(appData) == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, appName), nil
	}

	home, ok := lookupEnv("HOME")
	if !ok || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("HOME environment variable not set")
	}
	return filepath.Join(home, ".config", appName), nil
}

func targetPath(goos string, elements ...string) string {
	if goos != "windows" {
		return filepath.Join(elements...)
	}

	joined := strings.Join(elements, `\`)
	for strings.Contains(joined, `\\`) {
		joined = strings.ReplaceAll(joined, `\\`, `\`)
	}
	return joined
}

func posixPath(path string) string {
	return strings.ReplaceAll(path, `\`, "/")
}

// Install adds or updates a marker block in rcFile that sources installLines.
// If a marker block already exists, it is replaced.
func Install(rcFile string, installLines []string) error {
	if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
		return fmt.Errorf("creating rc directory %s: %w", filepath.Dir(rcFile), err)
	}

	existing, err := readRCFile(rcFile)
	if err != nil {
		return err
	}

	content := installedContent(existing, installLines)
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

func installBlock(installLines []string) string {
	return markerStart + "\n" + strings.Join(installLines, "\n") + "\n" + markerEnd + "\n"
}

func installedContent(existing string, installLines []string) string {
	if strings.Contains(existing, markerStart) {
		existing = removeMarkerBlock(existing)
	}

	trimmed := strings.TrimRight(existing, "\n")
	if trimmed == "" {
		return installBlock(installLines)
	}
	return trimmed + "\n" + installBlock(installLines)
}

func readRCFile(rcFile string) (string, error) {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading rc file %s: %w", rcFile, err)
	}
	return string(data), nil
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
