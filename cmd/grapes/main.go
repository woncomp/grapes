package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	errHelpRequested    = errors.New("help requested")
	outputPhases        = []string{shells.PhaseEnv, shells.PhaseMain, shells.PhaseSetup}
	defaultExecuteSetup = executeManagedSetup
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
	pathClean      string
	pathCleanMode  bool
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
	executeSetup   func(shells.Shell, string) error
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

	if opts.pathCleanMode {
		fmt.Println(cleanPathList(opts.pathClean, os.PathListSeparator))
		return
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
		case arg == "--path-clean":
			if i+1 >= len(args) {
				return cliOptions{}, fmt.Errorf("missing value for %s", arg)
			}
			i++
			opts.pathCleanMode = true
			opts.pathClean = args[i]
		case strings.HasPrefix(arg, "--path-clean="):
			opts.pathCleanMode = true
			opts.pathClean = strings.TrimPrefix(arg, "--path-clean=")
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

	if opts.pathCleanMode {
		if opts.masterPath != "" {
			return cliOptions{}, fmt.Errorf("--path-clean cannot be combined with an input file")
		}
		if len(opts.targets) > 0 || opts.assumeYes || opts.noLink || opts.dependencyMode != dependencyModePrompt {
			return cliOptions{}, fmt.Errorf("--path-clean cannot be combined with generation flags")
		}
		return opts, nil
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
	fmt.Fprintln(w, "   or: grapes --path-clean <path>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Generate shell rc files from local .grape fragments.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "      --path-clean path  Remove duplicate and empty PATH entries, print cleaned PATH, then exit")
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

func userHomeDir(goos string, lookupEnv func(string) (string, bool)) (string, error) {
	keys := []string{"HOME"}
	if goos == "windows" {
		keys = []string{"HOME", "USERPROFILE"}
	}
	for _, key := range keys {
		if value, ok := lookupEnv(key); ok && strings.TrimSpace(value) != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("home directory environment variable not set")
}

func ensureRuntimeDirs(goos string, lookupEnv func(string) (string, bool), outputDir string) error {
	home, err := userHomeDir(goos, lookupEnv)
	if err != nil {
		return err
	}

	dirs := []string{
		filepath.Join(outputDir, "cache"),
		filepath.Join(home, ".local", "state", "grapes"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating runtime directory %s: %w", dir, err)
		}
	}
	return nil
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

func cleanPathList(pathValue string, separator rune) string {
	parts := strings.Split(pathValue, string(separator))
	seen := make(map[string]bool, len(parts))
	cleaned := make([]string, 0, len(parts))

	for _, part := range parts {
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		cleaned = append(cleaned, part)
	}

	return strings.Join(cleaned, string(separator))
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
	executeSetup := opts.executeSetup
	if executeSetup == nil {
		executeSetup = defaultExecuteSetup
	}

	outputDir, err := managedOutputDir(opts.goos, lookupEnv)
	if err != nil {
		return err
	}
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving current executable path: %w", err)
	}

	if filepath.Ext(opts.masterPath) != ".toml" {
		return fmt.Errorf("%s is not a .toml file", opts.masterPath)
	}
	if err := ensureRuntimeDirs(opts.goos, lookupEnv, outputDir); err != nil {
		return err
	}

	grapesFile, err := parser.ParseGrapesFile(opts.masterPath)
	if err != nil {
		return err
	}

	if len(grapesFile.Imports) == 0 {
		return fmt.Errorf("master file has no imports")
	}

	fragDir := filepath.Dir(opts.masterPath)
	grapesHome, err := filepath.Abs(fragDir)
	if err != nil {
		return fmt.Errorf("resolving master directory %s: %w", fragDir, err)
	}
	grapes, err := parseAllGrapes(grapesFile.Imports)
	if err != nil {
		return err
	}

	sorted, err := resolver.Resolve(grapesFile.Imports, grapes)
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

	resolved := sorted
	var dependencyResults []grapeDependencyResult
	var dependencyResultsByGrape map[string]grapeDependencyResult
	allowedWarnings := false
	for {
		dependencyResults, err = checkGrapeDependencies(resolved, dependencyCheckOptions{lookupEnv: lookupEnv})
		if err != nil {
			return err
		}
		dependencyResultsByGrape = mapDependencyResultsByGrape(dependencyResults)

		allowWarnings := opts.dependencyMode == dependencyModeAllowWarnings
		fmt.Fprint(stdout, renderDependencyTable(dependencyResults, allowWarnings))
		action, err := ui.chooseDependencyAction(opts.dependencyMode, dependencyResults)
		if err != nil {
			return err
		}
		switch action {
		case dependencyActionRetry:
			continue
		case dependencyActionCancel:
			fmt.Fprintln(stdout, "Cancelled generation.")
			return nil
		case dependencyActionAllowWarnings:
			allowedWarnings = true
		default:
			allowedWarnings = false
		}
		break
	}

	sorted = filterRenderableGrapes(resolved, dependencyResults, allowedWarnings)

	type setupOutput struct {
		shell shells.Shell
		path  string
	}

	var outputs []writer.OutputFile
	var setupOutputs []setupOutput
	for _, target := range opts.targets {
		for _, phase := range outputPhases {
			var shellFragments []writer.Fragment
			hasGrapeFragments := false
			if phase == shells.PhaseEnv {
				injectedLines := preprocessor.InjectedEnvLines(target.Name(), outputDir, grapesHome)
				shellFragments = append(shellFragments, writer.Fragment{
					Name:    "__GRAPE_ENV",
					Content: strings.Join(injectedLines, "\n") + "\n",
				})
			}
			for _, f := range sorted {
				result := dependencyResultsByGrape[f.Key]
				for _, block := range f.Blocks {
					if block.Phase != phase {
						continue
					}
					rendered, err := renderer.RenderBlock(opts.goos, target.Name(), block.Env, block.Paths, block.Body)
					if err != nil {
						return fmt.Errorf("rendering %s for %s: %w", f.Label, target.Name(), err)
					}

					content, err := preprocessor.Process(rendered, target.Name())
					if err != nil {
						return fmt.Errorf("preprocessing %s for %s: %w", f.Label, target.Name(), err)
					}
					if strings.TrimSpace(content) == "" {
						continue
					}
					scopePrefix, err := renderGrapeScopePrefix(target.Name(), result)
					if err != nil {
						return fmt.Errorf("rendering grape scope for %s in %s: %w", f.Label, target.Name(), err)
					}
					shellFragments = append(shellFragments, writer.Fragment{
						Name:    f.Label,
						Content: scopePrefix + content,
					})
					hasGrapeFragments = true
				}
			}
			if phase == shells.PhaseSetup {
				if !hasGrapeFragments {
					continue
				}
				injectedLines := preprocessor.InjectedEnvLines(target.Name(), outputDir, grapesHome)
				shellFragments = append([]writer.Fragment{{
					Name:    "__GRAPE_ENV",
					Content: strings.Join(injectedLines, "\n") + "\n",
				}}, shellFragments...)
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
			if phase == shells.PhaseEnv || phase == shells.PhaseMain {
				pathCleanLine, err := preprocessor.PathCleanInjectionLine(target.Name(), execPath)
				if err != nil {
					return fmt.Errorf("rendering path cleanup for %s: %w", target.Name(), err)
				}
				shellFragments = append(shellFragments, writer.Fragment{
					Name:    "__PATH_CLEAN",
					Content: pathCleanLine + "\n",
				})
			}
			filename := target.ManagedFilename(phase)
			outputs = append(outputs, writer.OutputFile{
				Filename:  filename,
				Fragments: shellFragments,
			})
			if phase == shells.PhaseSetup {
				setupOutputs = append(setupOutputs, setupOutput{
					shell: target,
					path:  filepath.Join(outputDir, filename),
				})
			}
		}
	}

	if err := writer.Write(outputDir, outputs); err != nil {
		return err
	}
	for _, setupOutput := range setupOutputs {
		if err := executeSetup(setupOutput.shell, setupOutput.path); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Executed setup file %s\n", setupOutput.path)
	}

	generatedPaths := managedOutputPaths(outputDir, outputs)
	if opts.noLink {
		printSummary(stdout, generatedPaths, nil)
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
	printSummary(stdout, generatedPaths, linkReports)

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
		fmt.Fprintln(stdout, path)
	}
}

func printLinkFiles(stdout io.Writer, reports []linkReport) {
	fmt.Fprintln(stdout, "Linked files:")
	for _, report := range reports {
		fmt.Fprintln(stdout, report.path)
	}
}

func printSummary(stdout io.Writer, generatedPaths []string, linkReports []linkReport) {
	printGeneratedFiles(stdout, generatedPaths)
	if len(linkReports) == 0 {
		return
	}
	printLinkFiles(stdout, linkReports)
}

func filterRenderableGrapes(grapes []*parser.GrapeFile, results []grapeDependencyResult, allowWarnings bool) []*parser.GrapeFile {
	resultByName := make(map[string]grapeDependencyResult, len(results))
	for _, result := range results {
		resultByName[grapeIdentityKey(result.Grape)] = result
	}

	filtered := make([]*parser.GrapeFile, 0, len(grapes))
	for _, grape := range grapes {
		result, ok := resultByName[grapeIdentityKey(grape)]
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
		byGrape[grapeIdentityKey(result.Grape)] = result
	}
	return byGrape
}

func grapeIdentityKey(grape *parser.GrapeFile) string {
	if grape == nil {
		return ""
	}
	if strings.TrimSpace(grape.Key) != "" {
		return grape.Key
	}
	return grape.Name
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

func executeManagedSetup(shell shells.Shell, scriptPath string) error {
	var cmd *exec.Cmd
	switch shell.Name() {
	case "bash":
		cmd = exec.Command("bash", scriptPath)
	case "zsh":
		cmd = exec.Command("zsh", scriptPath)
	case "nushell":
		cmd = exec.Command("nu", scriptPath)
	case "pwsh":
		cmd = exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-File", scriptPath)
	default:
		return fmt.Errorf("unsupported setup shell %q", shell.Name())
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("executing setup file %s: %w", scriptPath, err)
		}
		return fmt.Errorf("executing setup file %s: %w\n%s", scriptPath, err, trimmed)
	}
	return nil
}

// parseAllGrapes loads the .grape files referenced by the master TOML file.
func parseAllGrapes(imports []parser.GrapeImport) ([]*parser.GrapeFile, error) {
	seen := make(map[string]bool)
	var grapes []*parser.GrapeFile
	for _, imp := range imports {
		if seen[imp.Key] {
			continue
		}
		seen[imp.Key] = true
		grape, err := parser.ParseGrapeFile(imp.ResolvedPath)
		if err != nil {
			return nil, err
		}
		grape.Label = imp.Label
		grape.Key = imp.Key
		grapes = append(grapes, grape)
	}
	return grapes, nil
}
