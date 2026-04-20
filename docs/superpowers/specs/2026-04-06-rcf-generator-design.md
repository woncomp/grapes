# Grapes: Current Shell RC Generator Design Spec

## Overview

Grapes is a Go CLI that generates managed shell rc files from composable fragments. Users point the CLI at a master `.grapes` file, Grapes loads the referenced `.grape` fragments, resolves dependencies, preprocesses shell-conditional content, writes managed files under `~/.config/grapes/`, and optionally links the user's rc files back to those managed outputs.

The current implementation targets `bash` and `zsh`. A run can generate one or more selected shells, defaults to the current shell when no target is specified, and can source either local fragments or built-in embedded fragments.

## File Model

There are two file types:

- **`.grape`** — reusable fragment files.
- **`.grapes`** — master files that declare the fragment import list and act as the CLI entry point.

Fragments are **multi-block** documents. Each block has optional YAML frontmatter and an associated body. A file without leading frontmatter is treated as a single `main` block.

### Block frontmatter

```yaml
---
deps: []          # first block only
imports: []       # master files only; first block only
phase: main       # env or main, default: main
env: {}           # environment variables to export before the body
paths: []         # PATH entries to prepend before the body
---
```

Field behavior:

- `deps` is only accepted on the first block of a fragment.
- `imports` is only meaningful on the first block of a master `.grapes` file.
- `phase` controls whether the block contributes to the managed `env` file or the managed `main rc` file.
- `env` expands into sorted `export KEY="VALUE"` lines before the body.
- `paths` expands into ordered `export PATH="<entry>:$PATH"` lines before the body.

Subsequent blocks may change `phase`, `env`, `paths`, and `body`, but not `deps` or `imports`.

### Example fragment

```yaml
---
deps:
  - path
phase: env
env:
  GOPATH: "${GOPATH:-$HOME/go}"
paths:
  - $GOPATH/bin
---

---
phase: main
#ifdef BASH
complete -C some-tool some-tool
#endif

#ifdef ZSH
autoload -Uz compinit && compinit
#endif
```

### Example master file

```yaml
---
imports:
  - go
  - prompt
  - aliases
---
```

## Fragment Sources

When Grapes resolves imports and dependencies, it loads fragments in this order:

1. A local `<name>.grape` file next to the master file.
2. A built-in embedded fragment from `fragments/*.grape` if no local file exists.

The repository currently embeds these curated fragments:

- `go`
- `nvm`
- `uv`
- `bun`
- `zoxide`
- `fzf`

This gives local projects an override path while still shipping useful defaults.

## Processing Pipeline

```text
master.grapes
  -> parser.ParseFile
  -> recursive fragment discovery (local first, embedded fallback)
  -> resolver.Resolve
  -> preprocessor.Process per selected shell and block
  -> writer.Write managed files into ~/.config/grapes/
  -> shells.Install link blocks unless --nolink
```

### Components

1. **CLI (`cmd/grapes`)**
   - Parses arguments.
   - Detects the current shell from `$SHELL` when `-t/--target` is omitted.
   - Requires a `.grapes` master file.
   - Orchestrates parsing, resolution, preprocessing, writing, and optional linking.

2. **Parser (`parser`)**
   - Parses `.grape` and `.grapes` files into `Fragment` values.
   - Supports multi-block documents.
   - Validates `phase` values and enforces first-block-only rules for `deps` and `imports`.
   - Expands first-class `env` and `paths` fields into shell code before preprocessing.

3. **Dependency resolver (`resolver`)**
   - Traverses imports plus transitive dependencies.
   - Produces a deterministic topological order.
   - Reports missing fragments and circular dependency paths.

4. **Preprocessor (`preprocessor`)**
   - Evaluates `#ifdef`, `#ifndef`, `#elif`, `#else`, and `#endif` directives.
   - Rejects unknown `#...` directives.
   - Injects `export __GRAPES_SHELL="<shell>"` at the top of each processed block.

5. **Writer (`writer`)**
   - Writes selected output files into `~/.config/grapes/`.
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

Supported directives:

- `#ifdef BASH`
- `#ifdef ZSH`
- `#ifndef BASH`
- `#ifndef ZSH`
- `#elif <shell>`
- `#else`
- `#endif`

Content outside directives is emitted for all selected shells.

Unknown `#...` lines are treated as invalid directives and produce an error with a line number.

## CLI Interface

### Basic usage

```bash
go run ./cmd/grapes <source.grapes>
```

### Explicit targets

```bash
go run ./cmd/grapes <source.grapes> -t zsh --target=bash
```

### Generate without linking rc files

```bash
go run ./cmd/grapes <source.grapes> --nolink
```

Behavior:

- `-t, --target` is repeatable.
- When no target is provided, Grapes detects the current shell from `$SHELL`.
- Managed outputs are always written to `~/.config/grapes/`.
- Link blocks are installed by default; `--nolink` skips rc file modification.
- Managed outputs for supported shells that were not selected in the current run are pruned.

## Managed Output Structure

For selected shells, Grapes writes:

```text
~/.config/grapes/
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
source "$HOME/.config/grapes/bashrc"
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
fragments/        embedded built-in `.grape` fragments
docs/superpowers/ design spec and as-built plan
```

## Validation

The repository validates this behavior with package-level Go tests across the CLI, parser, resolver, preprocessor, writer, shell integration, and embedded fragments:

```bash
go test ./...
```
