# Grapes Shell RC Generator As-Built Plan

> This document originally started as a forward-looking implementation checklist. It now serves as the reality-aligned execution record for the rc generator that exists in this repository.

## Status

- [x] Go module and CLI entry point are in place.
- [x] Fragment parsing, dependency resolution, preprocessing, output writing, and rc-file linking are implemented.
- [x] Built-in embedded fragments are shipped and tested.
- [x] End-to-end behavior is covered by the existing Go test suite.

## Goal Delivered

Grapes now reads a master `.grapes` file, recursively loads imported fragments and their dependencies, resolves them into a deterministic order, preprocesses each block for the selected shell targets, writes managed rc files under `~/.config/grapes/`, and links user rc files unless `--nolink` is requested.

## Final Architecture

```text
cmd/grapes/main.go   CLI orchestration
parser/              multi-block parsing with env/path expansion
resolver/            dependency graph traversal and topological ordering
preprocessor/        shell conditional evaluation
writer/              managed output file generation
shells/              shell-specific filenames and rc-file marker installation
fragments/           embedded default `.grape` fragments
```

## Delivered Workstreams

### 1. Bootstrap and CLI wiring

- [x] The executable entry point lives at `cmd/grapes/main.go`.
- [x] Argument parsing supports `-t/--target`, `--nolink`, and `-h/--help`.
- [x] When no explicit target is passed, the CLI detects the current shell from `$SHELL`.
- [x] The CLI requires a `.grapes` master file and a non-empty import list.

### 2. Parser implementation

- [x] `.grape` files are parsed as multi-block documents.
- [x] Blocks support `phase`, `env`, `paths`, and raw body content.
- [x] The first block may declare `deps`, and master files may declare `imports`.
- [x] `env` and `paths` are expanded into shell exports before preprocessing.
- [x] Invalid YAML and invalid phase values fail fast.

### 3. Dependency resolution

- [x] Imports and transitive dependencies are collected recursively.
- [x] Missing fragments produce explicit errors.
- [x] Topological sorting is deterministic.
- [x] Circular dependencies are reported with a readable cycle path.

### 4. Preprocessing

- [x] `#ifdef`, `#ifndef`, `#elif`, `#else`, and `#endif` are evaluated per shell.
- [x] Unknown `#...` directives are rejected.
- [x] `__GRAPES_SHELL` is injected into processed output for the active shell.

### 5. Managed output writing

- [x] Generated content is written to `~/.config/grapes/`.
- [x] Output is split by shell and phase into `bashenv`, `bashrc`, `zshenv`, and `zshrc`.
- [x] Only selected shell outputs are retained; stale managed files for other supported shells are pruned.

### 6. Rc-file linking

- [x] Linking is enabled by default and can be skipped with `--nolink`.
- [x] Marker blocks are installed idempotently in user rc files.
- [x] Bash env linking prefers `~/.bash_profile` and falls back to `~/.bashenv`.
- [x] Zsh linking targets `~/.zshenv` and `~/.zshrc`.

### 7. Embedded fragments

- [x] Built-in fragments are embedded with Go's `embed` support.
- [x] Local fragments override embedded fragments when both exist.
- [x] The repository currently ships `go`, `nvm`, `uv`, `bun`, `zoxide`, and `fzf` fragments.

### 8. Validation

- [x] Package-level tests cover the CLI, parser, resolver, preprocessor, writer, shells, and embedded fragments.
- [x] `go test ./...` passes for the current implementation.

## Notable Changes From The Original Draft Plan

The implementation evolved in a few important ways while the original checklist remained untouched:

- The old single-body fragment model became a multi-block format.
- `env` and `paths` became first-class frontmatter fields instead of requiring all shell setup to be handwritten in the body.
- The original `--lazy` wording was replaced by default-on linking with an explicit `--nolink` escape hatch.
- Rc-file installation logic lives in the `shells` package rather than a separate `lazy/` package.
- Built-in embedded fragments were added as part of the shipped product.
- The executable entry point lives under `cmd/grapes/`, matching the current repository layout.

## Current Validation Command

```bash
go test ./...
```

## Follow-up Guidance

This file should now be treated as historical implementation context. Any new feature work should start with a new spec and a new implementation plan rather than re-opening this one.
