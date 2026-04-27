# Depend File for NVM Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Add a `depend_file` dependency type with fuzzy file matching so `nvm.grape` can participate in dependency checks across Unix-style installs and Windows file layouts.

**Architecture:** Extend the `.grape` parser model to support an optional `depend_file` object alongside `depend_executable`. Reuse the existing dependency table and filtering flow by adding a new checker branch that expands `~`, expands environment variables, applies filepath globbing, accepts only matching files, and marks the grape `ok` if any pattern matches. Then configure `nvm.grape` with cross-platform candidate file patterns and add focused tests for parser, checker, embedded fragments, and end-to-end filtering.

**Tech Stack:** Go 1.26, standard library only, existing `parser`, `cmd/grapes`, embedded `fragments`, and Go package tests

---

## File Structure

### Existing files to modify

- `parser/parser.go`
- `parser/grape.go`
- `parser/grape_test.go`
- `cmd/grapes/dependencies.go`
- `cmd/grapes/dependencies_test.go`
- `cmd/grapes/main_test.go`
- `fragments/nvm.grape`
- `fragments/fragments_test.go`

### New files to create

- None

## Task 1: Add parser support for `depend_file`

- [ ] Write a failing test in `parser/grape_test.go` for parsing a `.grape` file with a valid `depend_file.paths` list.
- [ ] Run `go test ./parser -run TestParseGrapeFileDependFile` and verify it fails because the field/schema does not exist yet.
- [ ] Add a parser-facing `depend_file` config shape with `paths []string`.
- [ ] Add a runtime-facing `DependFile` field to `parser.GrapeFile`.
- [ ] Treat parser-time config mistakes as fatal errors with file context: missing `paths` or empty `paths`.
- [ ] Keep parse-time behavior literal: do not expand `~`, environment variables, or globs in the parser.
- [ ] Run `go test ./parser -run TestParseGrapeFileDependFile` and make it pass.

## Task 2: Add runtime file dependency checking

- [ ] Write a failing test in `cmd/grapes/dependencies_test.go` for a grape whose `depend_file` matches an existing file and returns `ok` with the matched file path.
- [ ] Run `go test ./cmd/grapes -run TestFileDependencyCheck` and verify it fails because the checker only understands executable dependencies.
- [ ] Extend the dependency result path so a grape may carry either `depend_executable`, `depend_file`, or neither.
- [ ] Implement `depend_file` checking: expand `~`, expand environment variables, apply filepath globbing, and scan matches for files only.
- [ ] Return `ok` when any candidate file pattern matches at least one file, with `LOCATION` set to the matched file and `VERSION=n/a`.
- [ ] Return `failed` when no candidate path pattern matches any file, with `LOCATION=not found`, `VERSION=n/a`, and a human-readable detail.
- [ ] Keep `depend_file` out of the warning flow; it should only produce `ok` or `failed`.
- [ ] Run the targeted dependency checker tests and keep them green.

## Task 3: Add tests for fuzzy path behavior and embedded nvm config

- [ ] Write failing dependency checker tests for `~` expansion, environment variable expansion, glob matching, and directory-vs-file rejection.
- [ ] Run `go test ./cmd/grapes -run 'TestFileDependencyCheck|TestExpandSearchPath'` and verify the new file-path tests fail for the intended reason.
- [ ] Implement the minimal checker changes needed to make those tests pass.
- [ ] Add a failing embedded fragment test in `fragments/fragments_test.go` asserting that `nvm.grape` now exposes `DependFile` and no longer relies on `DependExecutable`.
- [ ] Run `go test ./fragments -run TestEmbeddedBuiltinDependencyConfigs` and verify it fails before the fragment is updated.

## Task 4: Configure `nvm.grape` with cross-platform file candidates

- [ ] Update `fragments/nvm.grape` to include `depend_file.paths` entries that cover common Unix/macOS `nvm.sh` layouts and common Windows `nvm.exe` locations.
- [ ] Keep the fragment body unchanged; only add dependency metadata.
- [ ] Choose fuzzy patterns that are permissive but still file-based, using glob support where it meaningfully broadens detection.
- [ ] Run `go test ./fragments -run TestEmbeddedBuiltinDependencyConfigs` and make it pass.

## Task 5: Verify end-to-end filtering behavior for `depend_file`

- [ ] Write a failing end-to-end test in `cmd/grapes/main_test.go` where a grape with `depend_file` renders when a candidate file exists and is skipped when none exist.
- [ ] Run `go test ./cmd/grapes -run TestRunDependencyChecksFile` and verify it fails before the runtime integration is complete.
- [ ] Ensure the existing dependency table, prompt, and filtering flow treat `depend_file` exactly like other dependency results.
- [ ] Run `go test ./cmd/grapes`.

## Task 6: Final verification, commit, merge, and cleanup

- [ ] Run `gofmt -w parser/*.go cmd/grapes/*.go fragments/*.go`.
- [ ] Run `go test ./...`.
- [ ] Commit the spec, code, fragment, and tests with a focused message.
- [ ] Merge the worktree branch back into `main` locally.
- [ ] Run `go test ./...` again on merged `main`.
- [ ] Remove the worktree and delete the feature branch.

## Review Checklist

- [ ] `.grape` supports a single optional `depend_file` object with `paths`.
- [ ] `depend_file` supports `~`, environment variable expansion, and glob matching.
- [ ] `depend_file` only matches files, not directories.
- [ ] `depend_file` produces `ok` or `failed`, never `warning`.
- [ ] `nvm.grape` now has dependency metadata that works for common Unix/macOS and Windows installs.
- [ ] Existing `depend_executable` behavior remains unchanged.
- [ ] Full test suite passes before and after merge.
