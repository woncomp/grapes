package main

import (
	"strings"
	"testing"

	"github.com/woncomp/grapes/parser"
)

func TestDependencyTableRendersSummaryAndDetails(t *testing.T) {
	results := []grapeDependencyResult{
		{Grape: &parser.GrapeFile{Name: "plain"}, Dependency: "n/a", Status: dependencyStatusOK, Location: "n/a", Version: "n/a"},
		{Grape: &parser.GrapeFile{Name: "zoxide"}, Dependency: "executable:zoxide", Status: dependencyStatusWarning, Location: "/usr/bin/zoxide", Version: "unknown", Detail: "version output did not match version_regex"},
		{Grape: &parser.GrapeFile{Name: "tool"}, Dependency: "file", Status: dependencyStatusFailed, Location: "not found", Version: "n/a", Detail: "not installed"},
	}

	text := renderDependencyTable(results, false)
	for _, fragment := range []string{
		"GRAPE",
		"DEPENDENCY",
		"STATUS",
		"LOCATION",
		"VERSION",
		"RENDER",
		"plain",
		"zoxide",
		"warning",
		"failed",
		"unknown",
		"not found",
		"version output did not match version_regex",
		"not installed",
		"executable:zoxide",
		"file",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("table = %q, want fragment %q", text, fragment)
		}
	}
	if !strings.Contains(text, "zoxide") || !strings.Contains(text, "no") {
		t.Fatalf("table = %q, want warning render=no in safe mode", text)
	}
}

func TestDependencyTableMarksWarningsRenderableWhenIgnoringWarnings(t *testing.T) {
	results := []grapeDependencyResult{{
		Grape:      &parser.GrapeFile{Name: "zoxide"},
		Dependency: "executable:zoxide",
		Status:     dependencyStatusWarning,
		Location:   "/usr/bin/zoxide",
		Version:    "unknown",
	}}

	text := renderDependencyTable(results, true)
	if !strings.Contains(text, "yes") {
		t.Fatalf("table = %q, want render=yes when warnings are allowed", text)
	}
}
