package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/woncomp/grapes/shells"
)

func TestReviewShellDisplaysGroupedDiffsWithoutColor(t *testing.T) {
	plan := shellLinkPlan{
		shell: mustParseShell(t, "bash"),
		links: []reviewedLink{
			{review: shells.Review{RCFile: "/tmp/.bashrc", Current: "", Proposed: "# >>> grapes >>>\nsource \"/tmp/bashenv\"\nsource \"/tmp/bashrc\"\n# <<< grapes <<<\n", Changed: true}},
		},
	}

	var out bytes.Buffer
	ui := reviewUI{
		stdin:       strings.NewReader("y\n"),
		stdout:      &out,
		interactive: true,
	}

	approved, err := ui.reviewShell(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Fatal("approved = false, want true")
	}

	text := out.String()
	for _, fragment := range []string{"Reviewing bash", "/tmp/.bashrc", "Apply changes for bash? [y/N]:"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("output = %q, want fragment %q", text, fragment)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Fatalf("output unexpectedly contained ANSI color: %q", text)
	}
}

func TestReviewShellColorizesDiffWhenTTY(t *testing.T) {
	plan := shellLinkPlan{
		shell: mustParseShell(t, "bash"),
		links: []reviewedLink{{review: shells.Review{RCFile: "/tmp/.bashrc", Current: "", Proposed: "# >>> grapes >>>\nsource \"/tmp/bashrc\"\n# <<< grapes <<<\n", Changed: true}}},
	}

	var out bytes.Buffer
	ui := reviewUI{
		stdin:       strings.NewReader("y\n"),
		stdout:      &out,
		interactive: true,
		color:       true,
	}

	if _, err := ui.reviewShell(plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("output = %q, want ANSI color", out.String())
	}
}

func TestReviewShellFailsWhenPromptingNonInteractive(t *testing.T) {
	plan := shellLinkPlan{
		shell: mustParseShell(t, "bash"),
		links: []reviewedLink{{review: shells.Review{RCFile: "/tmp/.bashrc", Current: "", Proposed: "# >>> grapes >>>\nsource \"/tmp/bashrc\"\n# <<< grapes <<<\n", Changed: true}}},
	}

	ui := reviewUI{
		stdin:       strings.NewReader(""),
		stdout:      &bytes.Buffer{},
		interactive: false,
	}

	_, err := ui.reviewShell(plan)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--yes") || !strings.Contains(err.Error(), "--nolink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDependencyPromptAssumeYesUsesSafeMode(t *testing.T) {
	ui := reviewUI{assumeYes: true, stdout: &bytes.Buffer{}}

	decision, err := ui.chooseDependencyAction(dependencyModePrompt, []grapeDependencyResult{{Status: dependencyStatusWarning}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := decision, dependencyActionSafe; got != want {
		t.Fatalf("decision = %q, want %q", got, want)
	}
}

func TestDependencyPromptSafeModeSkipsPrompt(t *testing.T) {
	ui := reviewUI{stdout: &bytes.Buffer{}}

	decision, err := ui.chooseDependencyAction(dependencyModeSafe, []grapeDependencyResult{{Status: dependencyStatusWarning}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := decision, dependencyActionSafe; got != want {
		t.Fatalf("decision = %q, want %q", got, want)
	}
}

func TestDependencyPromptFailModeReturnsErrorOnIssues(t *testing.T) {
	ui := reviewUI{stdout: &bytes.Buffer{}}

	_, err := ui.chooseDependencyAction(dependencyModeFail, []grapeDependencyResult{{Status: dependencyStatusWarning}, {Status: dependencyStatusFailed}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dependency check failed") || !strings.Contains(err.Error(), "1 warning") || !strings.Contains(err.Error(), "1 failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDependencyPromptSupportsIgnoreWarnings(t *testing.T) {
	var out bytes.Buffer
	ui := reviewUI{
		stdin:       strings.NewReader("w\n"),
		stdout:      &out,
		interactive: true,
	}

	decision, err := ui.chooseDependencyAction(dependencyModePrompt, []grapeDependencyResult{{Status: dependencyStatusWarning}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := decision, dependencyActionAllowWarnings; got != want {
		t.Fatalf("decision = %q, want %q", got, want)
	}
	if !strings.Contains(out.String(), "continue safely") || !strings.Contains(out.String(), "ignore warnings") {
		t.Fatalf("output = %q, want warning options", out.String())
	}
}

func TestDependencyPromptWithoutWarningsUsesYesNo(t *testing.T) {
	ui := reviewUI{
		stdin:       strings.NewReader("y\n"),
		stdout:      &bytes.Buffer{},
		interactive: true,
	}

	decision, err := ui.chooseDependencyAction(dependencyModePrompt, []grapeDependencyResult{{Status: dependencyStatusOK}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := decision, dependencyActionSafe; got != want {
		t.Fatalf("decision = %q, want %q", got, want)
	}
}

func TestDependencyPromptFailsWhenNonInteractiveWithoutYes(t *testing.T) {
	ui := reviewUI{stdin: strings.NewReader(""), stdout: &bytes.Buffer{}, interactive: false}

	_, err := ui.chooseDependencyAction(dependencyModePrompt, []grapeDependencyResult{{Status: dependencyStatusWarning}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--yes") || !strings.Contains(err.Error(), "--dependency-mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}
