package shells

import (
	"fmt"
	"path/filepath"
)

type nushellShell struct{}

func init() {
	registerShell(nushellShell{})
}

func (nushellShell) Name() string { return "nushell" }

func (nushellShell) Aliases() []string { return []string{"nushell", "nu"} }

func (nushellShell) ManagedFilename(phase string) string {
	switch phase {
	case PhaseEnv:
		return "nushell-env.nu"
	default:
		return "nushell-config.nu"
	}
}

func (n nushellShell) LinkTargets(ctx TargetContext) ([]LinkTarget, error) {
	configDir, err := configDir(ctx.GOOS, ctx.LookupEnv, "nushell")
	if err != nil {
		return nil, err
	}
	return []LinkTarget{
		{
			RCFile: filepath.Join(configDir, "env.nu"),
			InstallLines: []string{
				fmt.Sprintf(`source-env "%s"`, targetPath(ctx.GOOS, ctx.OutputDir, n.ManagedFilename(PhaseEnv))),
			},
		},
		{
			RCFile: filepath.Join(configDir, "config.nu"),
			InstallLines: []string{
				fmt.Sprintf(`source "%s"`, targetPath(ctx.GOOS, ctx.OutputDir, n.ManagedFilename(PhaseMain))),
			},
		},
	}, nil
}
