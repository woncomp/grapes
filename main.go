package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/woncomp/grapes/lazy"
	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
	"github.com/woncomp/grapes/resolver"
	"github.com/woncomp/grapes/writer"
)

func main() {
	lazyFlag := flag.Bool("lazy", false, "also install source lines in system rc files")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: grapes <source.grapes> [--lazy]\n\n")
		fmt.Fprintf(os.Stderr, "Generate shell rc files from .grape fragments.\n\n")
		flag.PrintDefaults()
	}

	// Scan for --lazy in args before flag.Parse, since Go's flag package
	// stops parsing at the first non-flag argument.
	for _, arg := range os.Args[1:] {
		if arg == "--lazy" {
			*lazyFlag = true
		}
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	masterPath := flag.Arg(0)
	if err := run(masterPath, *lazyFlag); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(masterPath string, doLazy bool) error {
	// 1. Parse master file
	master, err := parser.ParseFile(masterPath)
	if err != nil {
		return err
	}

	if !master.IsMaster {
		return fmt.Errorf("%s is not a .grapes file", masterPath)
	}

	if len(master.Imports) == 0 {
		return fmt.Errorf("master file has no imports")
	}

	// 2. Parse all fragments in the same directory
	fragDir := filepath.Dir(masterPath)
	fragments, err := parseAllFragments(fragDir, master.Imports)
	if err != nil {
		return err
	}

	// 3. Resolve dependencies
	sorted, err := resolver.Resolve(master.Imports, fragments)
	if err != nil {
		return err
	}

	// 4. Preprocess per shell and write output
	outputDir := filepath.Join(os.Getenv("HOME"), ".config", "grapes")
	shells := []string{"bash", "zsh"}
	phases := []string{"env", "main"}

	var outputs []writer.ShellOutput
	for _, shell := range shells {
		for _, phase := range phases {
			var frags []writer.Fragment
			for _, f := range sorted {
				if f.Phase != phase {
					continue
				}
				content, err := preprocessor.Process(f.Body, shell)
				if err != nil {
					return fmt.Errorf("preprocessing %s for %s: %w", f.Name, shell, err)
				}
				frags = append(frags, writer.Fragment{
					Name:    f.Name,
					Content: content,
				})
			}
			outputs = append(outputs, writer.ShellOutput{
				Shell:     shell,
				Phase:     phase,
				Fragments: frags,
			})
		}
	}

	if err := writer.Write(outputDir, outputs); err != nil {
		return err
	}

	fmt.Printf("Generated rc files in %s\n", outputDir)

	// 5. Lazy install
	if doLazy {
		home := os.Getenv("HOME")
		if home == "" {
			return fmt.Errorf("HOME environment variable not set")
		}

		bashEnvTarget := lazy.DetectBashEnvTarget(home)
		installMap := map[string]string{
			bashEnvTarget:                     filepath.Join(outputDir, "bashenv"),
			filepath.Join(home, ".bashrc"):    filepath.Join(outputDir, "bashrc"),
			filepath.Join(home, ".zshenv"):    filepath.Join(outputDir, "zshenv"),
			filepath.Join(home, ".zshrc"):     filepath.Join(outputDir, "zshrc"),
		}

		for rcFile, sourcePath := range installMap {
			if err := lazy.Install(rcFile, sourcePath); err != nil {
				return fmt.Errorf("installing source in %s: %w", rcFile, err)
			}
			fmt.Printf("Installed source in %s\n", rcFile)
		}
	}

	return nil
}

// parseAllFragments recursively discovers and parses all .grape files
// reachable from the given import list.
func parseAllFragments(dir string, imports []string) ([]*parser.Fragment, error) {
	seen := make(map[string]bool)
	var fragments []*parser.Fragment

	var collect func(name string) error
	collect = func(name string) error {
		if seen[name] {
			return nil
		}
		seen[name] = true

		path := filepath.Join(dir, name+".grape")
		frag, err := parser.ParseFile(path)
		if err != nil {
			return err
		}
		fragments = append(fragments, frag)

		for _, dep := range frag.Deps {
			if err := collect(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, name := range imports {
		if err := collect(name); err != nil {
			return nil, err
		}
	}

	return fragments, nil
}
