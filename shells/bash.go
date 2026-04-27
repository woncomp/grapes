package shells

import (
	"fmt"
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

func (b bashShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
	home, err := homeDir(ctx.GOOS, ctx.LookupEnv)
	if err != nil {
		return nil, err
	}

	return []LinkTarget{
		{
			RCFile: detectBashEnvTarget(home),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, b.ManagedFilename(PhaseEnv)))),
			},
		},
		{
			RCFile: filepath.Join(home, ".bashrc"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, posixPath(filepath.Join(ctx.OutputDir, b.ManagedFilename(PhaseMain)))),
			},
		},
	}, nil
}

func detectBashEnvTarget(homeDir string) string {
	profile := filepath.Join(homeDir, ".bash_profile")
	if _, err := os.Stat(profile); err == nil {
		return profile
	}
	return filepath.Join(homeDir, ".bashenv")
}
