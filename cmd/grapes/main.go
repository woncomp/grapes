package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/woncomp/grapes/fragments"
	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
	"github.com/woncomp/grapes/resolver"
	"github.com/woncomp/grapes/shells"
	"github.com/woncomp/grapes/writer"
)

var (
	errHelpRequested = errors.New("help requested")
	outputPhases     = []string{shells.PhaseEnv, shells.PhaseMain}
)

type cliOptions struct {
	masterPath string
	targets    []shells.Shell
	noLink     bool
}

func main() {
	opts, err := parseArgs(os.Args[1:], os.LookupEnv)
	if err != nil {
		if errors.Is(err, errHelpRequested) {
			printUsage(os.Stdout)
			return
		}

		printUsage(os.Stderr)
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}

	if err := run(opts.masterPath, opts.targets, opts.noLink); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string, lookupEnv func(string) (string, bool)) (cliOptions, error) {
	var opts cliOptions
	seenTargets := make(map[string]bool)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-h" || arg == "--help":
			return cliOptions{}, errHelpRequested
		case arg == "--nolink":
			opts.noLink = true
		case arg == "-t" || arg == "--target":
			if i+1 >= len(args) {
				return cliOptions{}, fmt.Errorf("missing value for %s", arg)
			}
			i++
			if err := addTarget(&opts.targets, seenTargets, args[i]); err != nil {
				return cliOptions{}, err
			}
		case strings.HasPrefix(arg, "-t="):
			if err := addTarget(&opts.targets, seenTargets, strings.TrimPrefix(arg, "-t=")); err != nil {
				return cliOptions{}, err
			}
		case strings.HasPrefix(arg, "--target="):
			if err := addTarget(&opts.targets, seenTargets, strings.TrimPrefix(arg, "--target=")); err != nil {
				return cliOptions{}, err
			}
		case strings.HasPrefix(arg, "-"):
			return cliOptions{}, fmt.Errorf("unknown flag: %s", arg)
		default:
			if opts.masterPath != "" {
				return cliOptions{}, fmt.Errorf("unexpected extra argument: %s", arg)
			}
			opts.masterPath = arg
		}
	}

	if opts.masterPath == "" {
		return cliOptions{}, fmt.Errorf("missing input file")
	}

	if len(opts.targets) == 0 {
		target, err := shells.DetectCurrent(lookupEnv)
		if err != nil {
			return cliOptions{}, err
		}
		opts.targets = []shells.Shell{target}
	}

	return opts, nil
}

func addTarget(targets *[]shells.Shell, seen map[string]bool, raw string) error {
	target, err := shells.Parse(raw)
	if err != nil {
		return err
	}

	if seen[target.Name()] {
		return nil
	}

	seen[target.Name()] = true
	*targets = append(*targets, target)
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: grapes <input> [-t shell]... [--nolink]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Generate shell rc files from .grape fragments.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -t, --target shell   Target shell to generate and link (repeatable; default: current shell)")
	fmt.Fprintln(w, "      --nolink         Generate managed rc files only; do not link user rc files")
	fmt.Fprintln(w, "  -h, --help           Show help")
}

func run(masterPath string, targets []shells.Shell, noLink bool) error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

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

	fragDir := filepath.Dir(masterPath)
	frags, err := parseAllFragments(fragDir, master.Imports)
	if err != nil {
		return err
	}

	sorted, err := resolver.Resolve(master.Imports, frags)
	if err != nil {
		return err
	}

	outputDir := filepath.Join(home, ".config", "grapes")

	var outputs []writer.OutputFile
	for _, target := range targets {
		for _, phase := range outputPhases {
			var shellFragments []writer.Fragment
			for _, f := range sorted {
				for _, block := range f.Blocks {
					if block.Phase != phase {
						continue
					}
					content, err := preprocessor.Process(block.Body, target.Name())
					if err != nil {
						return fmt.Errorf("preprocessing %s for %s: %w", f.Name, target.Name(), err)
					}
					shellFragments = append(shellFragments, writer.Fragment{
						Name:    f.Name,
						Content: content,
					})
				}
			}
			outputs = append(outputs, writer.OutputFile{
				Filename:  target.ManagedFilename(phase),
				Fragments: shellFragments,
			})
		}
	}

	if err := writer.Write(outputDir, outputs); err != nil {
		return err
	}

	if err := pruneManagedOutputs(outputDir, targets); err != nil {
		return err
	}

	fmt.Printf("Generated rc files in %s for %s\n", outputDir, joinTargetNames(targets))

	if noLink {
		return nil
	}

	for _, target := range targets {
		for _, link := range target.LinkTargets(home, outputDir) {
			if err := shells.Install(link.RCFile, link.SourcePath); err != nil {
				return fmt.Errorf("installing source in %s: %w", link.RCFile, err)
			}
			fmt.Printf("Installed source in %s\n", link.RCFile)
		}
	}

	return nil
}

func joinTargetNames(targets []shells.Shell) string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name())
	}
	return strings.Join(names, ", ")
}

func pruneManagedOutputs(outputDir string, selectedTargets []shells.Shell) error {
	selected := make(map[string]bool)
	for _, target := range selectedTargets {
		for _, phase := range outputPhases {
			selected[target.ManagedFilename(phase)] = true
		}
	}

	for _, target := range shells.Supported() {
		for _, phase := range outputPhases {
			filename := target.ManagedFilename(phase)
			if selected[filename] {
				continue
			}
			path := filepath.Join(outputDir, filename)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing stale managed output %s: %w", path, err)
			}
		}
	}

	return nil
}

// parseAllFragments recursively discovers and parses all .grape files
// reachable from the given import list.
func parseAllFragments(dir string, imports []string) ([]*parser.Fragment, error) {
	seen := make(map[string]bool)
	var frags []*parser.Fragment

	var collect func(name string) error
	collect = func(name string) error {
		if seen[name] {
			return nil
		}
		seen[name] = true

		frag, err := parser.ParseFileOrEmbedded(dir, name, fragments.FS)
		if err != nil {
			return err
		}
		frags = append(frags, frag)

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

	return frags, nil
}
