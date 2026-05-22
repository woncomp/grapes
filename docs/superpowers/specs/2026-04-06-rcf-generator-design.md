# Grapes: Current Shell RC Generator Design Spec

## Overview

Grapes is a Go CLI that generates managed shell rc files from composable fragments. Users point the CLI at a master `.toml` file, Grapes loads the referenced `.grape` fragments, resolves dependencies, preprocesses shell-conditional content, writes managed files under `~/.local/state/grapes/`, and optionally links the user's rc files back to those managed outputs.

The current implementation targets `bash` and `zsh`. A run can generate one or more selected shells and defaults to the current shell when no target is specified.

## Grape file format reference

For `.grape` / master `.toml` authoring details, frontmatter semantics, phase guidance, dependency fields, and generated Grapes variables, see `docs/grapes/grape-file-reference.md`.

## Processing Pipeline

```text
docs/grapes.toml
  -> parser.ParseFile
  -> master-relative fragment resolution
  -> resolver.Resolve
  -> preprocessor.Process per selected shell and block
  -> writer.Write managed files into ~/.local/state/grapes/
  -> shells.Install link blocks unless --nolink
```

### Components

1. **CLI (`cmd/grapes`)**
   - Parses arguments.
   - Detects the current shell from `$SHELL` when `-t/--target` is omitted.
   - Requires a `.toml` master file.
   - Orchestrates parsing, resolution, preprocessing, writing, and optional linking.

2. **Parser (`parser`)**
   - Parses `.grape` fragments and master `.toml` files into `Fragment` values.
   - Supports multi-block documents.
   - Validates phase and frontmatter rules described in the grape authoring reference.

3. **Dependency resolver (`resolver`)**
   - Traverses imports plus transitive dependencies.
   - Produces a deterministic topological order.
   - Reports missing fragments and circular dependency paths.

4. **Preprocessor (`preprocessor`)**
   - Evaluates `--#ifdef`, `--#ifndef`, `--#elif`, `--#else`, and `--#endif` directives.
   - Rejects unknown `--#...` directives.
   - Injects generated Grapes environment variables into managed env output.

5. **Writer (`writer`)**
   - Writes selected output files into `~/.local/state/grapes/`.
   - Concatenates preprocessed fragment output in resolved order.

6. **Shell integration (`shells`)**
   - Models supported shells and their managed filenames.
   - Installs and removes marker-based source blocks in user rc files.
   - Decides shell-specific link targets, including bash's `.bash_profile` vs `.bashenv` behavior.

## Dependency Resolution Rules

- Dependencies are transitive.
- Only fragments reachable from the master's `imports` are processed.
- Missing fragments fail the run with an error.
- Cycles fail the run with a readable path such as `circular dependency: a -> b -> a`.
- Ordering is deterministic. Independent fragments and same-level edges are processed in lexicographic name order.

## Preprocessor Rules

Directive syntax and author-facing shell-condition guidance are documented in `docs/grapes/grape-file-reference.md`.

Unknown `--#...` lines are treated as invalid directives and produce an error with a line number.

## CLI Interface

### Basic usage

```bash
go run ./cmd/grapes ./docs/grapes.toml
```

### Explicit targets

```bash
go run ./cmd/grapes ./docs/grapes.toml -t zsh --target=bash
```

### Generate without linking rc files

```bash
go run ./cmd/grapes ./docs/grapes.toml --nolink
```

Behavior:

- `-t, --target` is repeatable.
- When no target is provided, Grapes detects the current shell from `$SHELL`.
- Managed outputs are always written to `~/.local/state/grapes/`.
- Link blocks are installed by default; `--nolink` skips rc file modification.
- Managed outputs for supported shells that were not selected in the current run are pruned.

## Managed Output Structure

For selected shells, Grapes writes:

```text
~/.local/state/grapes/
├── bashenv
├── bashrc
├── zshenv
└── zshrc
```

Phase mapping:

- `env` -> `<shell>env`
- `main` -> `<shell>rc`

## Link Installation

Installed marker block:

```bash
# >>> grapes >>>
source "$HOME/.local/state/grapes/bashrc"
# <<< grapes <<<
```

Link targets:

- `bashenv` -> `~/.bash_profile` if it exists, otherwise `~/.bashenv`
- `bashrc` -> `~/.bashrc`
- `zshenv` -> `~/.zshenv`
- `zshrc` -> `~/.zshrc`

Existing Grapes marker blocks are replaced in place so repeated runs stay idempotent.

## Error Handling

The current implementation surfaces explicit errors for:

- missing input file
- non-master input passed to the CLI
- master file with no imports
- missing fragment dependencies
- circular dependencies
- invalid YAML frontmatter
- invalid `phase` values
- malformed or unmatched preprocessor directives
- unsupported shell targets
- inability to detect the current shell from `$SHELL`
- missing `HOME`

## Repository Layout

```text
cmd/grapes/       CLI entry point and orchestration
parser/           multi-block fragment parsing
resolver/         dependency ordering and cycle detection
preprocessor/     shell directive evaluation
writer/           managed file output
shells/           shell metadata and rc-file linking
docs/             example `grapes.toml`
docs/grapes/      example `.grape` fragments
docs/superpowers/ design spec and as-built plan
```

## Validation

The repository validates this behavior with package-level Go tests across the CLI, parser, resolver, preprocessor, writer, shell integration, and repository example fragments:

```bash
go test ./...
```
