package shells

import (
	"fmt"
	"path/filepath"
)

type powershellShell struct{}

func init() {
	registerShell(powershellShell{})
}

func (powershellShell) Name() string { return "powershell" }

func (powershellShell) Aliases() []string { return []string{"powershell", "pwsh"} }

func (powershellShell) ManagedFilename(phase string) string {
	switch phase {
	case PhaseEnv:
		return "powershell-env.ps1"
	default:
		return "powershell-profile.ps1"
	}
}

func (p powershellShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
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
