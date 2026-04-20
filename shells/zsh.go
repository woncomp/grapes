package shells

import "path/filepath"

type zshShell struct{}

func init() {
	registerShell(zshShell{})
}

func (zshShell) Name() string {
	return "zsh"
}

func (zshShell) Aliases() []string {
	return []string{"zsh"}
}

func (zshShell) ManagedFilename(phase string) string {
	switch phase {
	case PhaseEnv:
		return "zshenv"
	default:
		return "zshrc"
	}
}

func (z zshShell) LinkTargets(home, outputDir string) []LinkTarget {
	return []LinkTarget{
		{
			RCFile:     filepath.Join(home, ".zshenv"),
			SourcePath: filepath.Join(outputDir, z.ManagedFilename(PhaseEnv)),
		},
		{
			RCFile:     filepath.Join(home, ".zshrc"),
			SourcePath: filepath.Join(outputDir, z.ManagedFilename(PhaseMain)),
		},
	}
}
