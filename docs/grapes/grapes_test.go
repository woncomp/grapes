package grapesdocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
)

var expectedFragments = []string{"go", "fnm", "uv", "bun", "zoxide", "fzf", "starship"}

func TestAllExampleFragmentsExist(t *testing.T) {
	for _, name := range expectedFragments {
		if _, err := os.Stat(filepath.Join(name + ".grape")); err != nil {
			t.Errorf("example fragment %s.grape not found: %v", name, err)
		}
	}
}

func TestExampleMasterFileExists(t *testing.T) {
	if _, err := os.Stat("master.grapes"); err != nil {
		t.Fatalf("master.grapes not found: %v", err)
	}
}

func TestExampleFragmentsValid(t *testing.T) {
	for _, name := range expectedFragments {
		t.Run(name, func(t *testing.T) {
			frag, err := parser.ParseGrapeFile(filepath.Join(name + ".grape"))
			if err != nil {
				t.Fatalf("ParseGrapeFile failed: %v", err)
			}

			if len(frag.Blocks) == 0 {
				t.Fatal("no blocks found")
			}

			for i, block := range frag.Blocks {
				if block.Phase != "env" && block.Phase != "main" && block.Phase != "setup" {
					t.Errorf("block %d: invalid phase %q", i, block.Phase)
				}

				hasContent := block.Body != "" || len(block.Env) > 0 || len(block.Paths) > 0
				if !hasContent {
					t.Errorf("block %d: has no content", i)
				}

				for _, shell := range []string{"bash", "zsh", "nushell", "pwsh"} {
					if _, err := preprocessor.Process(block.Body, shell); err != nil {
						t.Errorf("block %d: preprocessing for %s failed: %v", i, shell, err)
					}
				}
			}
		})
	}
}

func TestExampleFragmentDependencyConfigs(t *testing.T) {
	tests := []struct {
		name       string
		wantBinary string
		wantArgs   []string
		wantRegex  string
	}{
		{name: "bun", wantBinary: "bun", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`},
		{name: "fnm", wantBinary: "fnm", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`},
		{name: "fzf", wantBinary: "fzf", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`},
		{name: "go", wantBinary: "go", wantArgs: []string{"version"}, wantRegex: `go([0-9]+\.[0-9]+(?:\.[0-9]+)?)`},
		{name: "uv", wantBinary: "uv", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`},
		{name: "zoxide", wantBinary: "zoxide", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`},
		{name: "starship", wantBinary: "starship", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frag, err := parser.ParseGrapeFile(filepath.Join(tc.name + ".grape"))
			if err != nil {
				t.Fatal(err)
			}
			if frag.DependExecutable == nil {
				t.Fatal("DependExecutable = nil, want config")
			}
			if got, want := frag.DependExecutable.Binary, tc.wantBinary; got != want {
				t.Fatalf("Binary = %q, want %q", got, want)
			}
			if got, want := strings.Join(frag.DependExecutable.VersionArgs, ","), strings.Join(tc.wantArgs, ","); got != want {
				t.Fatalf("VersionArgs = %q, want %q", got, want)
			}
			if got, want := frag.DependExecutable.VersionRegex, tc.wantRegex; got != want {
				t.Fatalf("VersionRegex = %q, want %q", got, want)
			}
		})
	}
}

func TestFNMExampleEnvUsesScopedExecLocation(t *testing.T) {
	frag, err := parser.ParseGrapeFile("fnm.grape")
	if err != nil {
		t.Fatal(err)
	}

	envBody := fragmentBlockBody(t, frag, "env")
	for _, want := range []string{
		"GRAPES_EXEC_DIR",
		`export PATH="$GRAPES_EXEC_DIR:$PATH"`,
		"fnm env --shell",
	} {
		if !strings.Contains(envBody, want) {
			t.Fatalf("env block did not contain %q; got %q", want, envBody)
		}
	}
}

func TestFNMExampleUsesShellSpecificMainInit(t *testing.T) {
	frag, err := parser.ParseGrapeFile("fnm.grape")
	if err != nil {
		t.Fatal(err)
	}

	mainBody := fragmentBlockBody(t, frag, "main")
	for _, want := range []string{
		"fnm env --use-on-cd | Out-String | Invoke-Expression",
		`eval "$(fnm env --use-on-cd)"`,
	} {
		if !strings.Contains(mainBody, want) {
			t.Fatalf("main block did not contain %q; got %q", want, mainBody)
		}
	}
	for _, forbidden := range []string{
		"GRAPES_EXEC_DIR",
		"GRAPES_EXEC_PATH",
		"fnm env --json",
		"from json",
		"load-env",
		"FNM_MULTISHELL_PATH",
	} {
		if strings.Contains(mainBody, forbidden) {
			t.Fatalf("main block unexpectedly contained %q; got %q", forbidden, mainBody)
		}
	}
}

func TestZoxideExampleUsesCurrentShellSpecificInit(t *testing.T) {
	frag, err := parser.ParseGrapeFile("zoxide.grape")
	if err != nil {
		t.Fatal(err)
	}

	mainBody := fragmentBlockBody(t, frag, "main")
	for _, want := range []string{
		"~/.local/state/grapes/zoxide.ps1",
		`init $GRAPES_SHELL`,
		"source ~/.local/state/grapes/zoxide.nu",
	} {
		if !strings.Contains(mainBody, want) {
			t.Fatalf("main block did not contain %q; got %q", want, mainBody)
		}
	}

	setupBody := fragmentBlockBody(t, frag, "setup")
	for _, want := range []string{
		"init powershell",
		"& $env:GRAPES_EXEC_PATH",
		"Set-Content -Encoding utf8 -Path ~/.local/state/grapes/zoxide.ps1",
		"init nushell | save -f ~/.local/state/grapes/zoxide.nu",
	} {
		if !strings.Contains(setupBody, want) {
			t.Fatalf("setup block did not contain %q; got %q", want, setupBody)
		}
	}
}

func TestStarshipExampleUsesCachedShellSpecificInit(t *testing.T) {
	frag, err := parser.ParseGrapeFile("starship.grape")
	if err != nil {
		t.Fatal(err)
	}

	mainBody := fragmentBlockBody(t, frag, "main")
	for _, want := range []string{
		"~/.local/state/grapes/starship.ps1",
		"~/.local/state/grapes/starship.$GRAPES_SHELL",
		"source ~/.local/state/grapes/starship.nu",
	} {
		if !strings.Contains(mainBody, want) {
			t.Fatalf("main block did not contain %q; got %q", want, mainBody)
		}
	}

	setupBody := fragmentBlockBody(t, frag, "setup")
	for _, want := range []string{
		"init powershell",
		"Set-Content -Encoding utf8 -Path ~/.local/state/grapes/starship.ps1",
		"init nu | save -f ~/.local/state/grapes/starship.nu",
		"starship init $GRAPES_SHELL > ~/.local/state/grapes/starship.$GRAPES_SHELL",
	} {
		if !strings.Contains(setupBody, want) {
			t.Fatalf("setup block did not contain %q; got %q", want, setupBody)
		}
	}
}

func fragmentBlockBody(t *testing.T, frag *parser.GrapeFile, phase string) string {
	t.Helper()

	for _, block := range frag.Blocks {
		if block.Phase == phase {
			return block.Body
		}
	}

	t.Fatalf("phase %q not found", phase)
	return ""
}
