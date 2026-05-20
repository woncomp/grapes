package shells

import (
	"fmt"
	"path/filepath"
)

type pwshShell struct{}

func init() {
	registerShell(pwshShell{})
}

func (pwshShell) Name() string { return "pwsh" }

func (pwshShell) Aliases() []string { return []string{"pwsh"} }

func (pwshShell) ManagedFilename(phase string) string {
	switch phase {
	case PhaseEnv:
		return "pwsh-env.ps1"
	default:
		return "pwsh-profile.ps1"
	}
}

func (p pwshShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
	home, err := homeDir(ctx.GOOS, ctx.LookupEnv)
	if err != nil {
		return nil, err
	}

	profile := filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
	if ctx.GOOS == "windows" {
		profile = targetPath(ctx.GOOS, home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	}

	return []LinkTarget{
		{
			RCFile: profile,
			InstallLines: []string{
				fmt.Sprintf(`. "%s"`, targetPath(ctx.GOOS, ctx.OutputDir, p.ManagedFilename(PhaseEnv))),
				fmt.Sprintf(`. "%s"`, targetPath(ctx.GOOS, ctx.OutputDir, p.ManagedFilename(PhaseMain))),
			},
		},
	}, nil
}
