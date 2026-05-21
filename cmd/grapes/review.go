package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/woncomp/grapes/shells"
)

type reviewedLink struct {
	target shells.LinkTarget
	review shells.Review
}

type shellLinkPlan struct {
	shell shells.Shell
	links []reviewedLink
}

type reviewUI struct {
	stdin       io.Reader
	stdout      io.Writer
	interactive bool
	color       bool
	assumeYes   bool
	reader      *bufio.Reader
}

type dependencyAction string

const (
	dependencyActionCancel        dependencyAction = "cancel"
	dependencyActionSafe          dependencyAction = "safe"
	dependencyActionAllowWarnings dependencyAction = "allow_warnings"
	dependencyActionRetry         dependencyAction = "retry"
)

func previewShellLinkPlan(target shells.Shell, ctx shells.TargetContext) (shellLinkPlan, error) {
	links, err := target.LinkTargets(ctx)
	if err != nil {
		return shellLinkPlan{}, err
	}

	plan := shellLinkPlan{shell: target}
	for _, link := range links {
		review, err := shells.Preview(link.RCFile, link.InstallLines)
		if err != nil {
			return shellLinkPlan{}, err
		}
		plan.links = append(plan.links, reviewedLink{target: link, review: review})
	}
	return plan, nil
}

func (p shellLinkPlan) hasChanges() bool {
	for _, link := range p.links {
		if link.review.Changed {
			return true
		}
	}
	return false
}

func (ui *reviewUI) reviewShell(plan shellLinkPlan) (bool, error) {
	stdout := ui.stdout
	if stdout == nil {
		stdout = io.Discard
	}

	if !plan.hasChanges() {
		fmt.Fprintf(stdout, "No rc/profile changes for %s; skipping.\n", plan.shell.Name())
		return false, nil
	}

	fmt.Fprintf(stdout, "Reviewing %s\n", plan.shell.Name())
	for _, link := range plan.links {
		if !link.review.Changed {
			continue
		}
		diff := link.review.Diff()
		if ui.color {
			diff = colorizeDiff(diff)
		}
		fmt.Fprintf(stdout, "\n%s\n%s", link.review.RCFile, diff)
	}

	if ui.assumeYes {
		return true, nil
	}
	if !ui.interactive {
		return false, fmt.Errorf("shell link review requires an interactive terminal; rerun with --yes to accept changes automatically or --nolink to skip linking")
	}

	reader := ui.getReader()
	for {
		fmt.Fprintf(stdout, "Apply changes for %s? [y/N]: ", plan.shell.Name())
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		}
		fmt.Fprintln(stdout, "Please answer y or n.")
		if err == io.EOF {
			return false, nil
		}
	}
}

func dependencyActionForMode(mode dependencyMode, results []grapeDependencyResult) (dependencyAction, error) {
	hasIssues := false
	hasWarnings := false
	for _, result := range results {
		switch result.Status {
		case dependencyStatusWarning:
			hasWarnings = true
			hasIssues = true
		case dependencyStatusFailed:
			hasIssues = true
		}
	}

	if mode == "" {
		mode = dependencyModePrompt
	}

	switch mode {
	case dependencyModeSafe:
		return dependencyActionSafe, nil
	case dependencyModeAllowWarnings:
		return dependencyActionAllowWarnings, nil
	case dependencyModeFail:
		if hasIssues {
			warningCount, failedCount := dependencyIssueCounts(results)
			return dependencyActionCancel, fmt.Errorf("dependency check failed: %d warning, %d failed", warningCount, failedCount)
		}
		return dependencyActionSafe, nil
	case dependencyModePrompt:
		return dependencyActionCancel, nil
	default:
		if hasWarnings {
			_ = hasWarnings
		}
		return dependencyActionCancel, fmt.Errorf("unknown dependency mode: %s", mode)
	}
}

func dependencyIssueCounts(results []grapeDependencyResult) (warningCount, failedCount int) {
	for _, result := range results {
		switch result.Status {
		case dependencyStatusWarning:
			warningCount++
		case dependencyStatusFailed:
			failedCount++
		}
	}
	return warningCount, failedCount
}

func (ui *reviewUI) chooseDependencyAction(mode dependencyMode, results []grapeDependencyResult) (dependencyAction, error) {
	stdout := ui.stdout
	if stdout == nil {
		stdout = io.Discard
	}
	if mode != dependencyModePrompt {
		return dependencyActionForMode(mode, results)
	}
	if ui.assumeYes {
		return dependencyActionSafe, nil
	}
	if !ui.interactive {
		return dependencyActionCancel, fmt.Errorf("dependency review requires an interactive terminal; rerun with --yes or --dependency-mode to continue without prompting")
	}

	hasWarnings := false
	for _, result := range results {
		if result.Status == dependencyStatusWarning {
			hasWarnings = true
			break
		}
	}

	reader := ui.getReader()
	for {
		if hasWarnings {
			fmt.Fprintln(stdout, "Dependency check options: [y] continue safely, [w] ignore warnings, [r] retry check, [n] cancel")
			fmt.Fprint(stdout, "Choose action [y/w/r/N]: ")
		} else {
			fmt.Fprint(stdout, "Continue with generation? [y/r/N]: ")
		}
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return dependencyActionCancel, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes":
			return dependencyActionSafe, nil
		case "w":
			if hasWarnings {
				return dependencyActionAllowWarnings, nil
			}
		case "r":
			return dependencyActionRetry, nil
		case "", "n", "no":
			return dependencyActionCancel, nil
		}
		if hasWarnings {
			fmt.Fprintln(stdout, "Please answer y, w, r, or n.")
		} else {
			fmt.Fprintln(stdout, "Please answer y, r, or n.")
		}
		if err == io.EOF {
			return dependencyActionCancel, nil
		}
	}
}

func (ui *reviewUI) getReader() *bufio.Reader {
	if ui.reader == nil {
		ui.reader = bufio.NewReader(ui.stdin)
	}
	return ui.reader
}

func colorizeDiff(diff string) string {
	var b strings.Builder
	for _, line := range strings.SplitAfter(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			b.WriteString("\x1b[36m")
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		case strings.HasPrefix(line, "@@"):
			b.WriteString("\x1b[36m")
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		case strings.HasPrefix(line, "+"):
			b.WriteString("\x1b[32m")
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		case strings.HasPrefix(line, "-"):
			b.WriteString("\x1b[31m")
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		default:
			b.WriteString(line)
		}
	}
	return b.String()
}
