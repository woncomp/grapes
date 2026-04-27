# Builtin Executable Dependency Configs Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Add `depend_executable` defaults to the built-in grape fragments that map cleanly to real executables, leaving `nvm.grape` intentionally unconfigured.

**Architecture:** Reuse the existing `depend_executable` parser and runtime checker without changing CLI behavior or dependency semantics. Update only the built-in fragment definitions plus targeted tests that lock in the new embedded configuration values and document that `nvm.grape` remains outside the executable-only model.

**Tech Stack:** Go 1.26, embedded fragment files under `fragments/`, existing `parser` and `fragments` package tests

---

## File Structure

### Existing files to modify

- `fragments/bun.grape`
- `fragments/fzf.grape`
- `fragments/go.grape`
- `fragments/uv.grape`
- `fragments/fragments_test.go`
- `parser/grape_test.go`

### New files to create

- None

## Task 1: Add parser-level tests for representative builtin executable configs

- [ ] Write a failing test in `parser/grape_test.go` that parses a representative `bun`-style grape with `depend_executable` and asserts `binary`, `version_args`, and `version_regex` survive unchanged.
- [ ] Run `go test ./parser -run TestParseGrapeFileDependExecutable` and confirm the test still fails until the exact new expectations are met.
- [ ] Adjust the test or fixture content until it precisely captures the config shape you plan to add to built-in fragments.
- [ ] Run `go test ./parser -run TestParseGrapeFileDependExecutable` again and keep it green.

## Task 2: Add embedded fragment assertions for builtin dependency configs

- [ ] Write failing tests in `fragments/fragments_test.go` that parse the embedded `bun`, `fzf`, `go`, `uv`, `zoxide`, and `nvm` fragments.
- [ ] Assert that `bun`, `fzf`, `go`, `uv`, and `zoxide` each expose a non-nil `DependExecutable` with the expected `Binary` name.
- [ ] Assert that `nvm` exposes `DependExecutable == nil`.
- [ ] Assert tool-specific version command choices and regexes for the configured fragments.
- [ ] Run `go test ./fragments -run TestEmbeddedBuiltinDependencyConfigs` and verify it fails before editing the fragment files.

## Task 3: Add `depend_executable` blocks to builtin fragments

- [ ] Update `fragments/bun.grape` with `binary: bun` plus Bun-specific version command/regex.
- [ ] Update `fragments/fzf.grape` with `binary: fzf` plus fzf-specific version command/regex.
- [ ] Update `fragments/go.grape` with `binary: go` plus `go version` parsing.
- [ ] Update `fragments/uv.grape` with `binary: uv` plus uv-specific version command/regex.
- [ ] Keep `fragments/zoxide.grape` as-is unless a test proves its config shape needs a small correction.
- [ ] Leave `fragments/nvm.grape` unchanged and treat that as the intended design.
- [ ] Run `go test ./fragments -run TestEmbeddedBuiltinDependencyConfigs` and make it pass.

## Task 4: Run full verification and integrate cleanly

- [ ] Run `gofmt -w fragments/*.go parser/*.go` if any Go test files changed formatting.
- [ ] Run `go test ./...`.
- [ ] Commit only the builtin-config spec, fragment changes, and matching tests with a focused message.
- [ ] Merge the worktree branch back into `main` locally.
- [ ] Run `go test ./...` again on merged `main`.
- [ ] Remove the worktree and delete the feature branch.

## Review Checklist

- [ ] `bun.grape`, `fzf.grape`, `go.grape`, and `uv.grape` all have `depend_executable` configs.
- [ ] `zoxide.grape` remains configured.
- [ ] `nvm.grape` remains intentionally unconfigured.
- [ ] Embedded fragment tests verify the expected binary names and version command choices.
- [ ] No CLI/runtime behavior changed beyond the new built-in dependency metadata.
- [ ] Full test suite passes before and after merge.
