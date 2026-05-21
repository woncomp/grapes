package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

type dependencyMode string

const (
	dependencyModePrompt        dependencyMode = "prompt"
	dependencyModeSafe          dependencyMode = "safe"
	dependencyModeAllowWarnings dependencyMode = "allow-warnings"
	dependencyModeFail          dependencyMode = "fail"
)

type cliOptions struct {
	masterPath     string
	targets        []shells.Shell
	assumeYes      bool
	dependencyMode dependencyMode
	noLink         bool
}

type runOptions struct {
	masterPath     string
	targets        []shells.Shell
	assumeYes      bool
	dependencyMode dependencyMode
	noLink         bool
	lookupEnv      func(string) (string, bool)
	goos           string
	stdin          io.Reader
	stdout         io.Writer
	interactive    bool
	color          bool
}

type linkReport struct {
	status string
	path   string
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
		masterPath:     opts.masterPath,
		targets:        opts.targets,
		assumeYes:      opts.assumeYes,
		dependencyMode: opts.dependencyMode,
		noLink:         opts.noLink,
		lookupEnv:      os.LookupEnv,
		goos:           runtime.GOOS,
		stdin:          os.Stdin,
		stdout:         os.Stdout,
		interactive:    isCharDevice(os.Stdin) && isCharDevice(os.Stdout),
		color:          isCharDevice(os.Stdout),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string, lookupEnv func(string) (string, bool)) (cliOptions, error) {
	return parseArgsWithShellDetector(args, lookupEnv, shells.DetectCurrent)
}

func parseArgsWithShellDetector(args []string, lookupEnv func(string) (string, bool), detectShell func(func(string) (string, bool)) (shells.Shell, error)) (cliOptions, error) {
	opts := cliOptions{dependencyMode: dependencyModePrompt}
	seenTargets := make(map[string]bool)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-h" || arg == "--help":
			return cliOptions{}, errHelpRequested
		case arg == "-y" || arg == "--yes":
			opts.assumeYes = true
			opts.dependencyMode = dependencyModeSafe
		case strings.HasPrefix(arg, "--dependency-mode="):
			mode, err := parseDependencyMode(strings.TrimPrefix(arg, "--dependency-mode="))
			if err != nil {
				return cliOptions{}, err
			}
			opts.dependencyMode = mode
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
	if opts.assumeYes && opts.dependencyMode != dependencyModeSafe {
		return cliOptions{}, fmt.Errorf("--yes conflicts with --dependency-mode=%s", opts.dependencyMode)
	}

	if len(opts.targets) == 0 {
		target, err := detectShell(lookupEnv)
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
	fmt.Fprintln(w, "Usage: grapes <input> [-t shell]... [--dependency-mode mode] [--yes] [--nolink]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Generate shell rc files from local .grape fragments.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -t, --target shell   Target shell to generate and link (repeatable; default: current shell)")
	fmt.Fprintln(w, "      --dependency-mode mode  Dependency handling mode: prompt, safe, allow-warnings, fail")
	fmt.Fprintln(w, "  -y, --yes            Approve dependency review in safe mode and skip shell link prompts")
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
		masterPath:     masterPath,
		targets:        targets,
		dependencyMode: dependencyModePrompt,
		noLink:         noLink,
		lookupEnv:      os.LookupEnv,
		goos:           runtime.GOOS,
		stdin:          os.Stdin,
		stdout:         os.Stdout,
		interactive:    isCharDevice(os.Stdin) && isCharDevice(os.Stdout),
		color:          isCharDevice(os.Stdout),
	})
}

func parseDependencyMode(raw string) (dependencyMode, error) {
	mode := dependencyMode(strings.TrimSpace(raw))
	switch mode {
	case dependencyModePrompt, dependencyModeSafe, dependencyModeAllowWarnings, dependencyModeFail:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid value for --dependency-mode: %s", raw)
	}
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

	if opts.dependencyMode == "" {
		opts.dependencyMode = dependencyModePrompt
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
	dependencyResultsByGrape := mapDependencyResultsByGrape(dependencyResults)

	ui := reviewUI{
		stdin:       opts.stdin,
		stdout:      stdout,
		interactive: opts.interactive,
		color:       opts.color,
		assumeYes:   opts.assumeYes,
	}
	allowWarnings := opts.dependencyMode == dependencyModeAllowWarnings
	fmt.Fprint(stdout, renderDependencyTable(dependencyResults, allowWarnings))
	action, err := ui.chooseDependencyAction(opts.dependencyMode, dependencyResults)
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
			hasGrapeFragments := false
			if phase == shells.PhaseEnv {
				injectedLines := preprocessor.InjectedEnvLines(target.Name(), outputDir)
				shellFragments = append(shellFragments, writer.Fragment{
					Name:    "__GRAPE_ENV",
					Content: strings.Join(injectedLines, "\n") + "\n",
				})
			}
			for _, f := range sorted {
				result := dependencyResultsByGrape[f.Name]
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
					if strings.TrimSpace(content) == "" {
						continue
					}
					scopePrefix, err := renderGrapeScopePrefix(target.Name(), result)
					if err != nil {
						return fmt.Errorf("rendering grape scope for %s in %s: %w", f.Name, target.Name(), err)
					}
					shellFragments = append(shellFragments, writer.Fragment{
						Name:    f.Name,
						Content: scopePrefix + content,
					})
					hasGrapeFragments = true
				}
			}
			if hasGrapeFragments {
				scopeCleanup, err := renderer.RenderGrapeExecCleanup(target.Name())
				if err != nil {
					return fmt.Errorf("rendering grape scope cleanup for %s: %w", target.Name(), err)
				}
				shellFragments = append(shellFragments, writer.Fragment{
					Name:    "__GRAPE_SCOPE_CLEANUP",
					Content: scopeCleanup,
				})
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

	fmt.Fprintf(stdout, "Generated rc files in %s for %s\n", outputDir, joinTargetNames(opts.targets))
	printGeneratedFiles(stdout, managedOutputPaths(outputDir, outputs))

	if opts.noLink {
		return nil
	}

	ctx := shells.TargetContext{
		GOOS:      opts.goos,
		LookupEnv: lookupEnv,
		OutputDir: outputDir,
	}
	var linkReports []linkReport

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
			linkReports = append(linkReports, summarizeLinkPlan(plan, false)...)
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
		linkReports = append(linkReports, summarizeLinkPlan(plan, true)...)
	}
	printLinkFiles(stdout, linkReports)

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

func managedOutputPaths(outputDir string, outputs []writer.OutputFile) []string {
	paths := make([]string, 0, len(outputs))
	for _, output := range outputs {
		paths = append(paths, filepath.Join(outputDir, output.Filename))
	}
	return paths
}

func summarizeLinkPlan(plan shellLinkPlan, approved bool) []linkReport {
	reports := make([]linkReport, 0, len(plan.links))
	for _, link := range plan.links {
		status := "unchanged"
		if link.review.Changed {
			if approved {
				status = "linked"
			} else {
				status = "skipped"
			}
		}
		reports = append(reports, linkReport{
			status: status,
			path:   link.target.RCFile,
		})
	}
	return reports
}

func printGeneratedFiles(stdout io.Writer, paths []string) {
	fmt.Fprintln(stdout, "Generated files:")
	for _, path := range paths {
		fmt.Fprintf(stdout, "- %s\n", path)
	}
}

func printLinkFiles(stdout io.Writer, reports []linkReport) {
	fmt.Fprintln(stdout, "Linked files:")
	for _, report := range reports {
		fmt.Fprintf(stdout, "- %s %s\n", report.status, report.path)
	}
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

func mapDependencyResultsByGrape(results []grapeDependencyResult) map[string]grapeDependencyResult {
	byGrape := make(map[string]grapeDependencyResult, len(results))
	for _, result := range results {
		byGrape[result.Grape.Name] = result
	}
	return byGrape
}

func renderGrapeScopePrefix(shell string, result grapeDependencyResult) (string, error) {
	execPath, ok := grapeExecutableLocation(result)
	if !ok {
		return renderer.RenderGrapeExecCleanup(shell)
	}
	return renderer.RenderGrapeExecScope(shell, execPath, filepath.Dir(execPath), grapeExecutableVersion(result))
}

func grapeExecutableLocation(result grapeDependencyResult) (string, bool) {
	if result.Grape == nil || result.Grape.DependExecutable == nil {
		return "", false
	}
	path := strings.TrimSpace(result.Location)
	if path == "" || path == "n/a" || path == "not found" {
		return "", false
	}
	return path, true
}

func grapeExecutableVersion(result grapeDependencyResult) string {
	if result.Grape == nil || result.Grape.DependExecutable == nil {
		return ""
	}
	version := strings.TrimSpace(result.Version)
	if version == "" || version == "n/a" || version == "unknown" {
		return ""
	}
	return version
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
		grape, err := parser.ParseGrapeFile(filepath.Join(dir, name+".grape"))
		if err != nil {
			return nil, err
		}
		grapes = append(grapes, grape)
	}
	return grapes, nil
}
