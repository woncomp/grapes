package shells

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
	case PhaseSetup:
		return "zsh-setup"
	default:
		return "zshrc"
	}
}

func (z zshShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
	home, err := homeDir(ctx.GOOS, ctx.LookupEnv)
	if err != nil {
		return nil, err
	}
	zdotdir := home
	if value, ok := ctx.LookupEnv("ZDOTDIR"); ok && strings.TrimSpace(value) != "" {
		zdotdir = value
	}

	return []LinkTarget{
		{
			RCFile: filepath.Join(home, ".zshenv"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, z.ManagedFilename(PhaseEnv)))),
			},
		},
		{
			RCFile: filepath.Join(zdotdir, ".zshrc"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, z.ManagedFilename(PhaseMain)))),
			},
		},
	}, nil
}
