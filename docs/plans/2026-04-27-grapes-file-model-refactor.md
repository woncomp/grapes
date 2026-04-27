# Grapes and Grape File Model Refactor Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Split `.grapes` files and `.grape` files into separate parser data structures and entry points without changing any user-visible behavior.

**Architecture:** Replace the current shared `parser.Fragment` model with two explicit parse products: a `GrapesFile` for top-level imports and a `GrapeFile` for renderable blocks plus the current dependency metadata. Keep the existing CLI, resolver behavior, embedded fragment loading, render pipeline, and output semantics unchanged while moving call sites onto the new parser API.

**Tech Stack:** Go 1.26, standard library, existing `parser`, `resolver`, `cmd/grapes`, embedded `fragments`, and Go package tests

---

## File Structure

### Existing files to modify

- `parser/parser.go`
- `parser/parser_test.go`
- `resolver/resolver.go`
- `resolver/resolver_test.go`
- `cmd/grapes/main.go`
- `cmd/grapes/main_test.go`

### New files to create

- `parser/grapes.go`
- `parser/grapes_test.go`
- `parser/grape.go`
- `parser/grape_test.go`

## Task 1: Introduce separate parser models with compatibility-preserving behavior

- [ ] Create `parser/grapes.go` with a dedicated `GrapesFile` type that contains only `.grapes`-specific fields: path, name, and `Imports`.
- [ ] Create `parser/grape.go` with a dedicated `GrapeFile` type that contains the current `.grape`-specific fields: path, name, `Deps`, and `Blocks`.
- [ ] Move shared block/frontmatter parsing helpers out of the current shared structure so both parse paths can reuse delimiter handling, phase defaults, env parsing, path parsing, and body extraction.
- [ ] Add `ParseGrapesFile(path string) (*GrapesFile, error)` and `ParseGrapeFile(path string) (*GrapeFile, error)`.
- [ ] Add a `ParseEmbeddedGrape(dir, name string, embedFS embed.FS) (*GrapeFile, error)` helper to replace `ParseFileOrEmbedded` at the call sites that load built-in and local `.grape` files.
- [ ] Keep existing parse semantics unchanged: `.grapes` still only reads imports, `.grape` still supports the current multi-block frontmatter/body format, and block phase validation still matches the current rules.
- [ ] Remove the current shared `Fragment` fields that only exist to multiplex two file kinds (`Imports`, `IsMaster`) once all call sites are migrated, but preserve the existing `.grape` dependency metadata on `GrapeFile` for now so behavior does not change in this refactor.

## Task 2: Lock in parser behavior with focused TDD coverage

- [ ] Write a failing test in `parser/grapes_test.go` for parsing a `.grapes` file into `GrapesFile` with ordered `Imports`.
- [ ] Run `go test ./parser -run TestParseGrapesFile` and verify the new test fails for the expected missing API/type reason.
- [ ] Implement the minimal `GrapesFile` parser code to make that test pass.
- [ ] Write a failing test in `parser/grape_test.go` covering a `.grape` file with multiple blocks, structured `env`, structured `paths`, and raw body preservation.
- [ ] Run `go test ./parser -run TestParseGrapeFile` and verify the failure is due to the new API not existing or returning the wrong structure.
- [ ] Implement the minimal `GrapeFile` parser code to make that test pass.
- [ ] Add failing regression tests for: default `main` phase, invalid phase, unterminated frontmatter, no-frontmatter `.grape`, and embedded fallback/local override using the new parser entry points.
- [ ] Run `go test ./parser` and keep the parser package green.

## Task 3: Migrate resolver from dependency graph inputs to ordered grape lists with no behavior change

- [ ] Write a failing test in `resolver/resolver_test.go` that exercises the post-refactor API using `[]*parser.GrapeFile` instead of the old shared parser type while preserving current output ordering for imported grapes.
- [ ] Run `go test ./resolver -run TestResolve` and verify the test fails because the resolver signature and helpers still depend on the old parser type.
- [ ] Refactor `resolver.Resolve` so it accepts imported grape names plus `[]*parser.GrapeFile` and returns ordered `[]*parser.GrapeFile`.
- [ ] Preserve the current topological ordering semantics based on `.grape` `deps`, including cycle detection, missing dependency errors, and deterministic order for unrelated grapes.
- [ ] Remove resolver test helpers that fabricate deprecated parser structures and replace them with helpers that build `GrapeFile` values.
- [ ] Run `go test ./resolver`.

## Task 4: Migrate CLI loading and rendering to the new parser API

- [ ] Write a failing test in `cmd/grapes/main_test.go` that exercises the current CLI path through `run`/`runWithOptions` while asserting no user-visible behavior changes.
- [ ] Run `go test ./cmd/grapes -run TestRun` and verify the failure is caused by the parser API mismatch introduced by the refactor.
- [ ] Update `cmd/grapes/main.go` to call `parser.ParseGrapesFile` for the input `.grapes` file instead of the old generic parse function.
- [ ] Replace `parseAllFragments` with a loader that only loads the grapes named in `GrapesFile.Imports`, using the new embedded/local `.grape` parse helper.
- [ ] Update the render loop to iterate `[]*parser.GrapeFile` and keep the same per-block render/preprocess/write behavior.
- [ ] Preserve all existing error text that is still user-visible where practical, especially “not a .grapes file” and “master file has no imports”; rename the latter to “grapes file has no imports” only if tests and CLI usage are updated in the same change.
- [ ] Run `go test ./cmd/grapes`.

## Task 5: Remove obsolete shared-shape tests and run full verification

- [ ] Delete or rewrite old parser tests that assert `IsMaster` or the shared `Fragment` shape that no longer exists after the split, but keep coverage for `.grape` `Deps` behavior because this spec must not change functionality.
- [ ] Update any helper code or comments that still talk about “master” instead of “Grapes” when referring to the `.grapes` file structure.
- [ ] Run `gofmt -w parser/*.go resolver/*.go cmd/grapes/*.go`.
- [ ] Run `go test ./...`.

## Review Checklist

- [ ] `.grapes` and `.grape` parse into distinct exported Go types.
- [ ] `.grapes` no longer carries block/render fields in memory.
- [ ] `.grape` no longer carries import/master-only fields in memory, but still carries its current `deps` metadata until a later behavior-changing spec removes it.
- [ ] Embedded/local grape loading still works exactly as before.
- [ ] CLI behavior and generated output remain unchanged for existing inputs.
- [ ] All parser, resolver, and CLI tests pass after the refactor.
