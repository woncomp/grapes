package shells

import (
	"os"
	"path/filepath"
)

type bashShell struct{}

func init() {
	registerShell(bashShell{})
}

func (bashShell) Name() string {
	return "bash"
}

func (bashShell) Aliases() []string {
	return []string{"bash"}
}

func (bashShell) ManagedFilename(phase string) string {
	switch phase {
	case PhaseEnv:
		return "bashenv"
	default:
		return "bashrc"
	}
}

func (b bashShell) LinkTargets(home, outputDir string) []LinkTarget {
	return []LinkTarget{
		{
			RCFile:     detectBashEnvTarget(home),
			SourcePath: filepath.Join(outputDir, b.ManagedFilename(PhaseEnv)),
		},
		{
			RCFile:     filepath.Join(home, ".bashrc"),
			SourcePath: filepath.Join(outputDir, b.ManagedFilename(PhaseMain)),
		},
	}
}

func detectBashEnvTarget(homeDir string) string {
	profile := filepath.Join(homeDir, ".bash_profile")
	if _, err := os.Stat(profile); err == nil {
		return profile
	}
	return filepath.Join(homeDir, ".bashenv")
}
