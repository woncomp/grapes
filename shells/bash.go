package shells

import (
	"fmt"
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
	case PhaseSetup:
		return "bash-setup"
	default:
		return "bashrc"
	}
}

func (b bashShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
	home, err := homeDir(ctx.GOOS, ctx.LookupEnv)
	if err != nil {
		return nil, err
	}

	return []LinkTarget{
		{
			RCFile: filepath.Join(home, ".bashrc"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, b.ManagedFilename(PhaseEnv)))),
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, b.ManagedFilename(PhaseMain)))),
			},
		},
	}, nil
}
