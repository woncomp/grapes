package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/woncomp/grapes/fragments"
	"github.com/woncomp/grapes/parser"
	"github.com/woncomp/grapes/preprocessor"
	"github.com/woncomp/grapes/renderer"
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
	assumeYes  bool
	noLink     bool
}

type runOptions struct {
	masterPath  string
	targets     []shells.Shell
	assumeYes   bool
	noLink      bool
	lookupEnv   func(string) (string, bool)
	goos        string
	stdin       io.Reader
	stdout      io.Writer
	interactive bool
	color       bool
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

	if err := runWithOptions(runOptions{
		masterPath:  opts.masterPath,
		targets:     opts.targets,
		assumeYes:   opts.assumeYes,
		noLink:      opts.noLink,
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		interactive: isCharDevice(os.Stdin) && isCharDevice(os.Stdout),
		color:       isCharDevice(os.Stdout),
	}); err != nil {
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
		case arg == "-y" || arg == "--yes":
			opts.assumeYes = true
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
	fmt.Fprintln(w, "Usage: grapes <input> [-t shell]... [--yes] [--nolink]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Generate shell rc files from .grape fragments.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -t, --target shell   Target shell to generate and link (repeatable; default: current shell)")
	fmt.Fprintln(w, "  -y, --yes            Approve all shell rc/profile changes without prompting")
	fmt.Fprintln(w, "      --nolink         Generate managed rc files only; do not link user rc files")
	fmt.Fprintln(w, "  -h, --help           Show help")
}

func managedOutputDir(goos string, lookupEnv func(string) (string, bool)) (string, error) {
	if goos == "windows" {
		appData, ok := lookupEnv("APPDATA")
		if !ok || strings.TrimSpace(appData) == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "grapes"), nil
	}

	home, ok := lookupEnv("HOME")
	if !ok || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("HOME environment variable not set")
	}
	return filepath.Join(home, ".config", "grapes"), nil
}

func run(masterPath string, targets []shells.Shell, noLink bool) error {
	return runWithOptions(runOptions{
		masterPath:  masterPath,
		targets:     targets,
		noLink:      noLink,
		lookupEnv:   os.LookupEnv,
		goos:        runtime.GOOS,
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		interactive: isCharDevice(os.Stdin) && isCharDevice(os.Stdout),
		color:       isCharDevice(os.Stdout),
	})
}

func runWithOptions(opts runOptions) error {
	lookupEnv := opts.lookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	stdout := opts.stdout
	if stdout == nil {
		stdout = io.Discard
	}

	outputDir, err := managedOutputDir(opts.goos, lookupEnv)
	if err != nil {
		return err
	}

	if filepath.Ext(opts.masterPath) != ".grapes" {
		return fmt.Errorf("%s is not a .grapes file", opts.masterPath)
	}

	grapesFile, err := parser.ParseGrapesFile(opts.masterPath)
	if err != nil {
		return err
	}

	if len(grapesFile.Imports) == 0 {
		return fmt.Errorf("master file has no imports")
	}

	fragDir := filepath.Dir(opts.masterPath)
	grapes, err := parseAllGrapes(fragDir, grapesFile.Imports)
	if err != nil {
		return err
	}

	sorted, err := resolver.Resolve(grapesFile.Imports, grapes)
	if err != nil {
		return err
	}

	dependencyResults, err := checkGrapeDependencies(sorted, dependencyCheckOptions{lookupEnv: lookupEnv})
	if err != nil {
		return err
	}

	ui := reviewUI{
		stdin:       opts.stdin,
		stdout:      stdout,
		interactive: opts.interactive,
		color:       opts.color,
		assumeYes:   opts.assumeYes,
	}
	fmt.Fprint(stdout, renderDependencyTable(dependencyResults, false))
	action, err := ui.chooseDependencyAction(dependencyResults)
	if err != nil {
		return err
	}
	if action == dependencyActionCancel {
		fmt.Fprintln(stdout, "Cancelled generation.")
		return nil
	}

	allowedWarnings := action == dependencyActionAllowWarnings
	sorted = filterRenderableGrapes(sorted, dependencyResults, allowedWarnings)

	var outputs []writer.OutputFile
	for _, target := range opts.targets {
		for _, phase := range outputPhases {
			var shellFragments []writer.Fragment
			for _, f := range sorted {
				for _, block := range f.Blocks {
					if block.Phase != phase {
						continue
					}
					rendered, err := renderer.RenderBlock(opts.goos, target.Name(), block.Env, block.Paths, block.Body)
					if err != nil {
						return fmt.Errorf("rendering %s for %s: %w", f.Name, target.Name(), err)
					}

					content, err := preprocessor.Process(rendered, target.Name())
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

	if err := pruneManagedOutputs(outputDir, opts.targets); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Generated rc files in %s for %s\n", outputDir, joinTargetNames(opts.targets))

	if opts.noLink {
		return nil
	}

	ctx := shells.TargetContext{
		GOOS:      opts.goos,
		LookupEnv: lookupEnv,
		OutputDir: outputDir,
	}

	for _, target := range opts.targets {
		plan, err := previewShellLinkPlan(target, ctx)
		if err != nil {
			return fmt.Errorf("reviewing link targets for %s: %w", target.Name(), err)
		}

		approved, err := ui.reviewShell(plan)
		if err != nil {
			return err
		}
		if !approved {
			if plan.hasChanges() {
				fmt.Fprintf(stdout, "Skipped %s\n", target.Name())
			}
			continue
		}

		for _, link := range plan.links {
			if !link.review.Changed {
				continue
			}
			if err := shells.Install(link.target.RCFile, link.target.InstallLines); err != nil {
				return fmt.Errorf("installing source in %s: %w", link.target.RCFile, err)
			}
			fmt.Fprintf(stdout, "Installed source in %s\n", link.target.RCFile)
		}
	}

	return nil
}

func isCharDevice(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
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

func filterRenderableGrapes(grapes []*parser.GrapeFile, results []grapeDependencyResult, allowWarnings bool) []*parser.GrapeFile {
	resultByName := make(map[string]grapeDependencyResult, len(results))
	for _, result := range results {
		resultByName[result.Grape.Name] = result
	}

	filtered := make([]*parser.GrapeFile, 0, len(grapes))
	for _, grape := range grapes {
		result, ok := resultByName[grape.Name]
		if !ok {
			continue
		}
		switch result.Status {
		case dependencyStatusOK:
			filtered = append(filtered, grape)
		case dependencyStatusWarning:
			if allowWarnings {
				filtered = append(filtered, grape)
			}
		}
	}
	return filtered
}

// parseAllGrapes loads the named .grape files referenced by the .grapes file.
func parseAllGrapes(dir string, imports []string) ([]*parser.GrapeFile, error) {
	seen := make(map[string]bool)
	var grapes []*parser.GrapeFile
	for _, name := range imports {
		if seen[name] {
			continue
		}
		seen[name] = true
		grape, err := parser.ParseEmbeddedGrape(dir, name, fragments.FS)
		if err != nil {
			return nil, err
		}
		grapes = append(grapes, grape)
	}
	return grapes, nil
}
