# Shell Link Review Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make rc/profile linking review-first by default, showing grouped per-shell diffs with TTY-only color before modifying user-owned shell startup files.

**Architecture:** Keep actual rc/profile writes centralized in `shells.Install`, but extract the preview and diff logic into reusable helpers in `shells`. Add a small CLI review layer in `cmd/grapes` that groups pending rc/profile diffs per selected shell, applies ANSI colors only on TTY output, prompts once per shell, and then either installs or skips that shell as a unit.

**Tech Stack:** Go 1.26, standard library only, existing `cmd/grapes`, `shells`, and Go package tests

---

## File Structure

### Existing files to modify

- `cmd/grapes/main.go`
- `cmd/grapes/main_test.go`
- `shells/shells.go`

### New files to create

- `shells/review.go`
- `shells/review_test.go`
- `cmd/grapes/review.go`
- `cmd/grapes/review_test.go`

## Task 1: Implement shell link review end-to-end

- [ ] Update `cmd/grapes/main.go` to make review-first linking the default, add `-y/--yes`, and keep `--nolink` unchanged.
- [ ] Refactor `shells/shells.go` so install content generation is reusable for both previewing and writing.
- [ ] Add preview support in `shells/review.go`: current content, proposed content, changed/no-op detection, and plain unified diff output.
- [ ] Add CLI review helpers in `cmd/grapes/review.go`: grouped per-shell display, TTY-only color, yes/no prompt handling, and non-interactive failure behavior.
- [ ] Wire the run loop so each shell is previewed once, approved or rejected once, then installed or skipped as a unit.
- [ ] Add focused tests in `shells/review_test.go` and `cmd/grapes/review_test.go`.
- [ ] Add end-to-end coverage in `cmd/grapes/main_test.go` for approve, reject, unchanged, `--yes`, and non-interactive behavior.
- [ ] Run `go test ./...`.

## Review Checklist

- [ ] Default behavior is review-first.
- [ ] `--yes` bypasses prompts but still skips unchanged targets.
- [ ] `--nolink` still bypasses linking entirely.
- [ ] Review is grouped per selected shell, not per file.
- [ ] Only user-owned rc/profile link targets are shown in the review.
- [ ] Diff output is colorized only on TTYs and plain otherwise.
- [ ] Non-interactive prompting fails clearly with guidance to use `--yes` or `--nolink`.
