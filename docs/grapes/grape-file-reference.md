# Grapes file authoring reference

This document is the single source of truth for authoring and documenting `.grape` fragment files and master `.toml` files in this repository.

Repository examples live in `docs/grapes/`, with the example master file at `docs/grapes.toml`. Master-file imports resolve relative to the input master file, so these checked-in files are examples and local starting points rather than embedded runtime defaults.

## File types

- **`.grape`**: reusable fragment files.
- **`.toml`**: master files that declare the fragment import list and act as the CLI entry point.

## `.grape` block model

Fragments are multi-block documents. Each block has optional YAML frontmatter plus a body. A file without leading frontmatter is treated as a single `main` block.

```yaml
---
phase: main
---
echo hello
```

### Block frontmatter

```yaml
---
deps: []               # first block only
phase: main            # env, main, or setup; default: main
env: {}                # environment variables rendered before the body
paths: []              # PATH entries prepended before the body
depend_executable: {}  # optional executable dependency check
depend_file: {}        # optional file dependency check
---
```

Field behavior:

- `deps` is only accepted on the first block of a fragment.
- `phase` controls whether the block contributes to the managed `env` file or the managed `main` file.
- `env` renders as shell-native environment assignments before the body.
- `paths` renders as shell-native PATH prepend operations before the body.
- `depend_executable` declares an executable that must be found for the fragment to render.
- `depend_file` declares one or more file patterns that must match for the fragment to render.

Subsequent blocks may change `phase`, `env`, `paths`, and `body`, but not `deps`.

## Master `.toml` file model

Master files use an ordered TOML array of `[[grape]]` tables:

```toml
[[grape]]
import = "go"

[[grape]]
import = "fnm"

[[grape]]
from = "shared"
import = "prompt.grape"
```

Behavior:

- each `[[grape]]` entry is order-sensitive
- `import` is required and may be extensionless or an explicit `.grape` relative path
- `from` is optional; when omitted, Grapes resolves the import from the same directory as the master `.toml` file
- `from` and `import` both support subdirectories plus `.` and `..`
- duplicate imports that normalize to the same fragment path are de-duplicated after the first occurrence
- only reachable imported fragments are processed

## Phases

Grapes has two phases:

- `env`
- `main`
- `setup`

Authoring guidance:

- Prefer `env` for environment variables, PATH setup, and environment state that later commands depend on.
- Reserve `main` for interactive shell behavior such as completions, aliases, prompts, and other non-environment startup logic.
- Use `setup` for one-time initialization work that should run automatically right after generation but should not be linked into shell startup files.
- When a fragment needs both phases, keep blocks ordered as `main` first and `env` second. Do not put `env` before `main`.
- `setup` outputs are generated per target shell, executed once immediately after generation, and never linked into rc/profile files.

## Structured `env` and `paths`

Structured frontmatter stays shell-agnostic in source files and is rendered natively per target shell during generation.

Example:

```yaml
---
phase: env
env:
  GOPATH: "${GOPATH:-$HOME/go}"
paths:
  - $GOPATH/bin
---
```

Rendering behavior:

- `pwsh`: `$env:KEY = ...`, `$env:PATH = ... + [System.IO.Path]::PathSeparator + $env:PATH`
- `nushell`: `$env.KEY = ...`, `$env.PATH = ($env.PATH | prepend ...)`
- `zsh` / `bash`: `export KEY=...`, `export PATH=...:$PATH`

## Shell conditionals

Fragment bodies may use preprocessor directives:

- `#ifdef <shell>`
- `#ifndef <shell>`
- `#elif <shell>`
- `#else`
- `#endif`

Supported canonical shell names:

- `pwsh`
- `nushell`
- `zsh`
- `bash`

Common examples:

- `#ifdef PWSH`
- `#ifdef NUSHELL`
- `#ifdef ZSH`
- `#ifdef BASH`

## Dependency-gated fragments

### `depend_executable`

Use `depend_executable` when a fragment should render only if a tool executable is available.

```yaml
---
phase: main
depend_executable:
  binary: fnm
  search_paths:
    - ~/.local/bin
  version_args:
    - --version
  version_regex: "([0-9]+\\.[0-9]+\\.[0-9]+)"
---
```

Behavior:

- `binary` is required
- `search_paths` is optional
- `version_regex` requires `version_args`
- if the executable is missing, the fragment is skipped

### `depend_file`

Use `depend_file` when a fragment should render only if one of several file patterns exists.

```yaml
---
phase: main
depend_file:
  paths:
    - ~/.tool/tool.sh
    - $HOME/.config/tool/*.nu
---
```

Behavior:

- `paths` is required
- patterns may use `~`, environment variables, and globs
- if no pattern matches, the fragment is skipped

## Generated Grapes environment variables

Generated `env` outputs inject:

- `GRAPES_SHELL`: the canonical target shell name
- `GRAPES_HOME`: the directory that contains the master `.toml` file
- `GRAPES_OUTPUT_PATH`: the managed output directory that contains the generated files
- `GRAPES_OUT_CACHE_DIR`: the `cache` subdirectory under `GRAPES_OUTPUT_PATH`

During generation, the `grapes` executable creates both `GRAPES_OUT_CACHE_DIR` and `~/.local/state/grapes` before writing managed outputs. Generated shell files no longer emit mkdir logic for those directories.

Executable-gated fragment scopes also inject:

- `GRAPES_EXEC_PATH`: the resolved executable path for the current grape scope
- `GRAPES_EXEC_DIR`: the parent directory of `GRAPES_EXEC_PATH`
- `GRAPES_EXEC_VERSION`: the detected version string for the current executable dependency when available

These scoped executable variables are set at the start of each rendered grape scope and cleaned up in the generated file cleanup section.

## Minimal examples

### Fragment with both phases

```yaml
---
phase: main
#ifdef BASH
complete -C some-tool some-tool
#endif

#ifdef ZSH
autoload -Uz compinit && compinit
#endif

---
phase: env
env:
  GOPATH: "${GOPATH:-$HOME/go}"
paths:
  - $GOPATH/bin
---
```

### Master file

```toml
[[grape]]
import = "go"

[[grape]]
import = "fnm"

[[grape]]
import = "zoxide"
```
