package fragments

import (
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

			frag, err := parser.ParseString(name, string(data))
			if err != nil {
				t.Fatalf("ParseString failed: %v", err)
			}

			if frag.Phase != "env" && frag.Phase != "main" {
				t.Errorf("invalid phase %q", frag.Phase)
			}
			if frag.Body == "" {
				t.Error("body is empty")
			}

			// Must preprocess without errors for both shells
			for _, shell := range []string{"bash", "zsh"} {
				_, err := preprocessor.Process(frag.Body, shell)
				if err != nil {
					t.Errorf("preprocessing for %s failed: %v", shell, err)
				}
			}
		})
	}
}
