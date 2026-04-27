package shells

import (
	"fmt"
	"path/filepath"
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
	default:
		return "zshrc"
	}
}

func (z zshShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
	home, err := homeDir(ctx.GOOS, ctx.LookupEnv)
	if err != nil {
		return nil, err
	}

	return []LinkTarget{
		{
			RCFile: filepath.Join(home, ".zshenv"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, z.ManagedFilename(PhaseEnv)))),
			},
		},
		{
			RCFile: filepath.Join(home, ".zshrc"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, z.ManagedFilename(PhaseMain)))),
			},
		},
	}, nil
}
