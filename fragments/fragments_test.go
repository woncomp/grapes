package fragments

import (
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
)

var expectedFragments = []string{"go", "nvm", "uv", "bun", "zoxide", "fzf"}

func TestAllFragmentsEmbedded(t *testing.T) {
	for _, name := range expectedFragments {
		_, err := FS.ReadFile(name + ".grape")
		if err != nil {
			t.Errorf("embedded fragment %s.grape not found: %v", name, err)
		}
	}
}

func TestEmbeddedFragmentsValid(t *testing.T) {
	for _, name := range expectedFragments {
		t.Run(name, func(t *testing.T) {
			data, err := FS.ReadFile(name + ".grape")
			if err != nil {
				t.Fatal(err)
			}

			frag, err := parser.ParseGrapeString(name, string(data), "<embedded:"+name+">")
			if err != nil {
				t.Fatalf("ParseGrapeString failed: %v", err)
			}

			if len(frag.Blocks) == 0 {
				t.Fatal("no blocks found")
			}

			for i, block := range frag.Blocks {
				if block.Phase != "env" && block.Phase != "main" {
					t.Errorf("block %d: invalid phase %q", i, block.Phase)
				}

				hasContent := block.Body != "" || len(block.Env) > 0 || len(block.Paths) > 0
				if !hasContent {
					t.Errorf("block %d: has no content", i)
				}

				for _, shell := range []string{"bash", "zsh"} {
					out, err := preprocessor.Process(block.Body, shell)
					if err != nil {
						t.Errorf("block %d: preprocessing for %s failed: %v", i, shell, err)
					}
					if !strings.Contains(out, `__GRAPES_SHELL="`+shell+`"`) {
						t.Errorf("block %d: preprocessor should inject __GRAPES_SHELL for %s", i, shell)
					}
				}
			}
		})
	}
}

func TestEmbeddedBuiltinDependencyConfigs(t *testing.T) {
	tests := []struct {
		name           string
		wantBinary     string
		wantArgs       []string
		wantRegex      string
		wantConfigured bool
	}{
		{name: "bun", wantBinary: "bun", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`, wantConfigured: true},
		{name: "fzf", wantBinary: "fzf", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`, wantConfigured: true},
		{name: "go", wantBinary: "go", wantArgs: []string{"version"}, wantRegex: `go([0-9]+\.[0-9]+(?:\.[0-9]+)?)`, wantConfigured: true},
		{name: "uv", wantBinary: "uv", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`, wantConfigured: true},
		{name: "zoxide", wantBinary: "zoxide", wantArgs: []string{"--version"}, wantRegex: `([0-9]+\.[0-9]+\.[0-9]+)`, wantConfigured: true},
		{name: "nvm", wantConfigured: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := FS.ReadFile(tc.name + ".grape")
			if err != nil {
				t.Fatal(err)
			}
			frag, err := parser.ParseGrapeString(tc.name, string(data), "<embedded:"+tc.name+">")
			if err != nil {
				t.Fatal(err)
			}

			if !tc.wantConfigured {
				if frag.DependExecutable != nil {
					t.Fatalf("DependExecutable = %#v, want nil", frag.DependExecutable)
				}
				return
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
