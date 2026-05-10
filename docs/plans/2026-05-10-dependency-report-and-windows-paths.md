# Dependency Report Labels and Windows Path Expansion Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Make dependency reports explicitly show dependency type and broaden Windows executable fallback search to cover more common install locations.

**Architecture:** Keep the existing dependency checker and table flow, but enrich dependency result metadata so the table can render `executable:<name>` and `file` distinctly. Extend the Windows branch of common executable search paths with a small, high-signal set of additional directories that cover common package manager, toolchain, and installed-program scenarios without changing non-Windows behavior.

**Tech Stack:** Go 1.26, standard library only, existing `cmd/grapes` dependency checker and table tests

---

## File Structure

### Existing files to modify

- `cmd/grapes/dependencies.go`
- `cmd/grapes/dependencies_test.go`
- `cmd/grapes/dependency_table.go`
- `cmd/grapes/dependency_table_test.go`

### New files to create

- `docs/plans/2026-05-10-dependency-report-and-windows-paths.md`

## Task 1: Add failing tests for dependency labels

- [ ] Write a failing test in `cmd/grapes/dependency_table_test.go` asserting executable dependencies render as `executable:<binary>` and file dependencies render as `file`.
- [ ] Run `go test ./cmd/grapes -run TestDependencyTable` and confirm the label expectation fails before implementation.
- [ ] Implement the minimal data/formatting changes to make the dependency column type-aware.
- [ ] Run `go test ./cmd/grapes -run TestDependencyTable` again.

## Task 2: Add failing tests for broader Windows paths

- [ ] Write failing tests in `cmd/grapes/dependencies_test.go` covering additional Windows fallback directories such as `%ProgramFiles%`, `%ProgramFiles(x86)%`, `%LOCALAPPDATA%\\Programs`, `%USERPROFILE%\\.cargo\\bin`, and `%USERPROFILE%\\.dotnet\\tools`.
- [ ] Run `go test ./cmd/grapes -run TestExecutableDependencyCheckWindows` and confirm the new cases fail before the path list is expanded.
- [ ] Implement the minimal Windows search-path additions needed to make those tests pass while preserving existing behavior.
- [ ] Run `go test ./cmd/grapes -run TestExecutableDependencyCheckWindows` again.

## Task 3: Full verification and integration

- [ ] Run `gofmt -w cmd/grapes/*.go`.
- [ ] Run `go test ./...`.
- [ ] Commit the spec and implementation with a focused message.
- [ ] Merge the branch back into `main` locally.
- [ ] Run `go test ./...` on merged `main`.
- [ ] Push `main` to `origin`.
- [ ] Remove the worktree and delete the feature branch.

## Review Checklist

- [ ] Dependency tables distinguish executable and file dependencies clearly.
- [ ] Non-dependent grapes still render `n/a`.
- [ ] Windows fallback lookup covers more common installation roots.
- [ ] Non-Windows lookup behavior remains unchanged.
- [ ] Full test suite passes before merge and after push.
