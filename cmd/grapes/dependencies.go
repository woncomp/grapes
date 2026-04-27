package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/woncomp/grapes/parser"
)

type dependencyStatus string

const (
	dependencyStatusOK      dependencyStatus = "ok"
	dependencyStatusWarning dependencyStatus = "warning"
	dependencyStatusFailed  dependencyStatus = "failed"
)

type grapeDependencyResult struct {
	Grape      *parser.GrapeFile
	Dependency string
	Status     dependencyStatus
	Location   string
	Version    string
	Detail     string
}

type dependencyCheckOptions struct {
	lookupEnv  func(string) (string, bool)
	lookPath   func(string) (string, error)
	pathExists func(string) bool
	runCommand func(string, ...string) (string, error)
}

func checkGrapeDependencies(grapes []*parser.GrapeFile, opts dependencyCheckOptions) ([]grapeDependencyResult, error) {
	if opts.lookupEnv == nil {
		opts.lookupEnv = os.LookupEnv
	}
	if opts.lookPath == nil {
		opts.lookPath = exec.LookPath
	}
	if opts.pathExists == nil {
		opts.pathExists = func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && !info.IsDir()
		}
	}
	if opts.runCommand == nil {
		opts.runCommand = func(path string, args ...string) (string, error) {
			out, err := exec.Command(path, args...).CombinedOutput()
			return string(out), err
		}
	}

	results := make([]grapeDependencyResult, 0, len(grapes))
	for _, grape := range grapes {
		result, err := checkGrapeDependency(grape, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func checkGrapeDependency(grape *parser.GrapeFile, opts dependencyCheckOptions) (grapeDependencyResult, error) {
	if grape.DependExecutable == nil && grape.DependFile == nil {
		return grapeDependencyResult{
			Grape:      grape,
			Dependency: "n/a",
			Status:     dependencyStatusOK,
			Location:   "n/a",
			Version:    "n/a",
		}, nil
	}
	if grape.DependFile != nil {
		return checkGrapeFileDependency(grape, opts)
	}

	dep := grape.DependExecutable
	path, ok := findExecutable(dep, opts)
	if !ok {
		return grapeDependencyResult{
			Grape:      grape,
			Dependency: dep.Binary,
			Status:     dependencyStatusFailed,
			Location:   "not found",
			Version:    "n/a",
			Detail:     fmt.Sprintf("executable %q was not found in PATH, common paths, or configured search_paths", dep.Binary),
		}, nil
	}

	result := grapeDependencyResult{
		Grape:      grape,
		Dependency: dep.Binary,
		Status:     dependencyStatusOK,
		Location:   path,
		Version:    "n/a",
	}
	if len(dep.VersionArgs) == 0 || dep.VersionRegex == "" {
		return result, nil
	}

	output, err := opts.runCommand(path, dep.VersionArgs...)
	if err != nil {
		result.Status = dependencyStatusWarning
		result.Version = "unknown"
		result.Detail = fmt.Sprintf("version command failed: %v", err)
		return result, nil
	}

	re := regexp.MustCompile(dep.VersionRegex)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		result.Status = dependencyStatusWarning
		result.Version = "unknown"
		result.Detail = "version output did not match version_regex"
		return result, nil
	}
	result.Version = matches[1]
	return result, nil
}

func checkGrapeFileDependency(grape *parser.GrapeFile, opts dependencyCheckOptions) (grapeDependencyResult, error) {
	matches := findMatchingFiles(grape.DependFile.Paths, opts.lookupEnv)
	if len(matches) == 0 {
		return grapeDependencyResult{
			Grape:      grape,
			Dependency: "file",
			Status:     dependencyStatusFailed,
			Location:   "not found",
			Version:    "n/a",
			Detail:     "no configured depend_file patterns matched an existing file",
		}, nil
	}
	return grapeDependencyResult{
		Grape:      grape,
		Dependency: "file",
		Status:     dependencyStatusOK,
		Location:   matches[0],
		Version:    "n/a",
	}, nil
}

func findExecutable(dep *parser.DependExecutable, opts dependencyCheckOptions) (string, bool) {
	if path, err := opts.lookPath(dep.Binary); err == nil && strings.TrimSpace(path) != "" {
		return path, true
	}

	for _, dir := range commonExecutableSearchPaths(opts.lookupEnv) {
		candidate := filepath.Join(dir, dep.Binary)
		if opts.pathExists(candidate) {
			return candidate, true
		}
	}
	for _, dir := range expandSearchPaths(dep.SearchPaths, opts.lookupEnv) {
		candidate := filepath.Join(dir, dep.Binary)
		if opts.pathExists(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func commonExecutableSearchPaths(lookupEnv func(string) (string, bool)) []string {
	paths := []string{"/usr/local/bin", "/opt/homebrew/bin", "/usr/bin"}
	if home, ok := lookupEnv("HOME"); ok && strings.TrimSpace(home) != "" {
		paths = append(paths, filepath.Join(home, ".local", "bin"))
	}
	return paths
}

func findMatchingFiles(patterns []string, lookupEnv func(string) (string, bool)) []string {
	var matches []string
	seen := make(map[string]bool)
	for _, pattern := range expandSearchPaths(patterns, lookupEnv) {
		matched, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, path := range matched {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			if !seen[path] {
				seen[path] = true
				matches = append(matches, path)
			}
		}
	}
	sort.Strings(matches)
	return matches
}

func expandSearchPaths(paths []string, lookupEnv func(string) (string, bool)) []string {
	expanded := make([]string, 0, len(paths))
	for _, path := range paths {
		expandedPath := path
		if strings.HasPrefix(expandedPath, "~/") {
			if home, ok := lookupEnv("HOME"); ok && strings.TrimSpace(home) != "" {
				expandedPath = filepath.Join(home, expandedPath[2:])
			}
		}
		expandedPath = os.Expand(expandedPath, func(key string) string {
			if value, ok := lookupEnv(key); ok {
				return value
			}
			return ""
		})
		expanded = append(expanded, expandedPath)
	}
	return expanded
}

func isNotFound(err error) bool {
	return err != nil && errors.Is(err, exec.ErrNotFound)
}
