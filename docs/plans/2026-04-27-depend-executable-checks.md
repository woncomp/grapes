# Executable Dependency Checks Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Add optional `depend_executable` checks to `.grape` files, show a dependency status table before generation, and render only grapes the user chooses to allow.

**Architecture:** Extend the `.grape` parser model with a single optional `depend_executable` configuration object and remove grape-to-grape dependency behavior so `.grapes` becomes the only file that selects which grapes participate. Add an executable dependency checker that inspects each imported grape before rendering, classifies results as `ok`, `warning`, or `failed`, prints a summary table plus details, asks the user how to continue, and then filters the grape set before the existing render/write/link pipeline runs.

**Tech Stack:** Go 1.26, standard library only, existing `cmd/grapes`, `parser`, embedded `fragments`, and Go package tests

---

## File Structure

### Existing files to modify

- `parser/grape.go`
- `parser/grape_test.go`
- `parser/grapes.go`
- `parser/grapes_test.go`
- `resolver/resolver.go`
- `resolver/resolver_test.go`
- `cmd/grapes/main.go`
- `cmd/grapes/main_test.go`
- `cmd/grapes/review.go`
- `cmd/grapes/review_test.go`
- `fragments/zoxide.grape`

### New files to create

- `cmd/grapes/dependencies.go`
- `cmd/grapes/dependencies_test.go`
- `cmd/grapes/dependency_table.go`
- `cmd/grapes/dependency_table_test.go`

## Task 1: Extend `.grape` parsing with `depend_executable`

- [ ] Add a `DependExecutable` field to `parser.GrapeFile` that holds the optional executable dependency configuration for one grape.
- [ ] Define a parser-facing config struct with fields: `Binary string`, `SearchPaths []string`, `VersionArgs []string`, and `VersionRegex string`.
- [ ] Keep the field optional; grapes without `depend_executable` must parse successfully and behave exactly as they do today.
- [ ] Treat parser-time config mistakes as fatal errors with file context: missing `binary`, invalid regex, or `version_regex` present without `version_args`.
- [ ] Ensure `~` and environment variables are not expanded during parse time; preserve raw configuration values for the runtime checker.

## Task 2: Add parser TDD coverage for the new schema

- [ ] Write a failing test in `parser/grape_test.go` for parsing a valid `depend_executable` block into the new config struct.
- [ ] Run `go test ./parser -run TestParseGrapeFileDependExecutable` and verify it fails because the field/schema does not exist yet.
- [ ] Implement the minimal parser changes to make that test pass.
- [ ] Write failing tests for each fatal config error: missing `binary`, invalid `version_regex`, and `version_regex` without `version_args`.
- [ ] Run the targeted parser tests, confirm they fail for the intended reason, then implement the minimal validation logic.
- [ ] Run `go test ./parser`.

## Task 3: Remove grape-to-grape dependencies so only `.grapes` selects grapes

- [ ] Write a failing test in `parser/grape_test.go` that confirms `.grape` no longer accepts the old `deps` field once this feature ships.
- [ ] Run `go test ./parser -run TestParseGrapeFileRejectsDeps` and verify it fails before the implementation change.
- [ ] Remove the old `.grape` dependency metadata from `parser.GrapeFile` and update parser validation so `deps` is no longer part of the supported `.grape` schema.
- [ ] Write a failing test in `resolver/resolver_test.go` for the new simpler API that preserves `.grapes` import order instead of topological sorting by grape dependencies.
- [ ] Run `go test ./resolver -run TestResolveImportsInOrder` and verify it fails before the resolver is simplified.
- [ ] Simplify `resolver.Resolve` so it validates that each imported grape exists and returns grapes in the `.grapes` import order with no transitive dependency expansion.
- [ ] Update CLI tests and fixtures that currently rely on grape-to-grape `deps` behavior.
- [ ] Run `go test ./parser ./resolver`.

## Task 4: Build the executable dependency checker

- [ ] Create `cmd/grapes/dependencies.go` with runtime result types for one grape and one executable check outcome.
- [ ] Model statuses explicitly as `ok`, `warning`, and `failed`.
- [ ] Implement executable lookup in this fixed order: `PATH`, built-in common paths, then configured `search_paths`.
- [ ] Add helpers to expand `~` and environment variables in runtime `search_paths` using the current environment lookup function.
- [ ] If the binary cannot be found, return `failed` with `LOCATION=not found`, `VERSION=n/a`, and a human-readable detail.
- [ ] If the binary is found and no version settings are configured, return `ok` with the resolved executable path and `VERSION=n/a`.
- [ ] If the binary is found and version settings are configured, execute `<binary-path> + version_args`, capture combined output, and apply `version_regex`.
- [ ] If version command execution fails or regex extraction fails, return `warning` with the resolved location, `VERSION=unknown`, and a detail string explaining why.
- [ ] If version extraction succeeds, return `ok` with both location and parsed version.

## Task 5: Add checker tests before wiring it into the CLI

- [ ] Write a failing targeted test in `cmd/grapes/dependencies_test.go` for PATH lookup success without version checks.
- [ ] Run `go test ./cmd/grapes -run TestExecutableDependencyCheck` and verify the failure is due to missing checker code.
- [ ] Implement the minimal checker code to make that test pass.
- [ ] Add failing tests for built-in path fallback, configured `search_paths` fallback, missing binary (`failed`), version command success, version command execution error (`warning`), and regex miss (`warning`).
- [ ] Add tests for `~` and environment variable expansion in configured `search_paths`.
- [ ] Keep tests hermetic by creating temporary executables/scripts in temp directories rather than depending on host-installed tools.
- [ ] Run `go test ./cmd/grapes -run 'TestExecutableDependencyCheck|TestExpandSearchPath'`.

## Task 6: Render dependency status as a table with detailed notes

- [ ] Create `cmd/grapes/dependency_table.go` to format a summary table with the columns `GRAPE`, `DEPENDENCY`, `STATUS`, `LOCATION`, `VERSION`, and `RENDER`.
- [ ] In the summary table, use explicit placeholders: `not found`/`n/a` for location and `unknown`/`n/a` for version.
- [ ] Print one row per imported grape, including grapes without `depend_executable`.
- [ ] For grapes without a dependency config, show `DEPENDENCY=n/a`, `STATUS=ok`, `LOCATION=n/a`, `VERSION=n/a`, and `RENDER=yes`.
- [ ] After the summary table, print per-grape detail lines only for `warning` and `failed` results.
- [ ] Add tests in `cmd/grapes/dependency_table_test.go` covering mixed `ok`/`warning`/`failed` output, stable column alignment, and explicit placeholder values.
- [ ] Run `go test ./cmd/grapes -run TestDependencyTable`.

## Task 7: Add pre-generation confirmation flow with warning override

- [ ] Extend `cmd/grapes/review.go` or a new helper to prompt after the dependency table is shown.
- [ ] Support three outcomes in interactive mode: cancel, continue safely (render only `ok`), and continue while ignoring warnings (render `ok + warning`).
- [ ] Define the exact prompt text in tests first so the CLI interaction is locked down before implementation.
- [ ] Make `--yes` behave as “continue safely” by default: render `ok`, skip `warning`, and never prompt.
- [ ] If there are no `warning` results, collapse the prompt to a simpler continue/cancel flow while preserving the same safe semantics.
- [ ] If running non-interactively without `--yes`, fail clearly with guidance that `--yes` continues without prompting.
- [ ] Add focused tests in `cmd/grapes/review_test.go` for the new dependency confirmation choices and non-interactive behavior.
- [ ] Run `go test ./cmd/grapes -run 'TestReview|TestDependencyPrompt'`.

## Task 8: Filter grapes before render/write/link and update fixtures

- [ ] Write a failing end-to-end test in `cmd/grapes/main_test.go` where one imported grape is `ok`, one is `warning`, and one is `failed`, and assert the rendered output changes according to the chosen confirmation mode.
- [ ] Run the targeted end-to-end test and verify it fails before implementation.
- [ ] Wire dependency checking into `runWithOptions` after the imported grapes are loaded and before any rendering occurs.
- [ ] Print the dependency table before any shell link review output.
- [ ] Build the allowed grape list from the confirmation decision: `ok` only for safe mode and `ok + warning` for ignore-warning mode.
- [ ] Leave `failed` grapes excluded from rendering in all modes.
- [ ] Ensure shell link review still runs after dependency filtering and still respects `--nolink`.
- [ ] Update `fragments/zoxide.grape` to include a realistic `depend_executable` example.
- [ ] Add end-to-end coverage in `cmd/grapes/main_test.go` for: no dependency config, `ok`, `warning` skipped, `warning` allowed, `failed` skipped, `--yes`, and non-interactive without `--yes`.
- [ ] Run `go test ./cmd/grapes`.

## Task 9: Final verification

- [ ] Run `gofmt -w parser/*.go cmd/grapes/*.go fragments/*.grape` if any Go files or test fixtures changed formatting-sensitive content.
- [ ] Run `go test ./...`.

## Review Checklist

- [ ] `.grape` supports a single optional `depend_executable` object.
- [ ] `.grape` no longer supports grape-to-grape `deps`; `.grapes` import order is now the only selection mechanism.
- [ ] Parser config errors fail fast before any dependency status table is shown.
- [ ] Executable lookup order is PATH, built-in paths, then configured `search_paths`.
- [ ] Version problems produce `warning`, not fatal parser/runtime failure.
- [ ] `--yes` approves generation but still excludes warning grapes unless the user explicitly chooses to ignore warnings in interactive mode.
- [ ] Failed executable dependencies are always skipped from rendering.
- [ ] Grapes without dependency config still render normally.
- [ ] Dependency status is displayed before any rc files are written or linked.
