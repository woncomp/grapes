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
}

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

func (ui reviewUI) reviewShell(plan shellLinkPlan) (bool, error) {
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

	reader := bufio.NewReader(ui.stdin)
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
