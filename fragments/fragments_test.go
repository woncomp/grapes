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

				// Must preprocess without errors for both shells
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
