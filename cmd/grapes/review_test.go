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
			{review: shells.Review{RCFile: "/tmp/.bashenv", Current: "", Proposed: "# >>> grapes >>>\nsource \"/tmp/bashenv\"\n# <<< grapes <<<\n", Changed: true}},
			{review: shells.Review{RCFile: "/tmp/.bashrc", Current: "", Proposed: "# >>> grapes >>>\nsource \"/tmp/bashrc\"\n# <<< grapes <<<\n", Changed: true}},
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
	for _, fragment := range []string{"Reviewing bash", "/tmp/.bashenv", "/tmp/.bashrc", "Apply changes for bash? [y/N]:"} {
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
