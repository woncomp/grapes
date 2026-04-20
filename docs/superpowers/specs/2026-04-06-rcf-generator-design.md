# Grapes: Shell RC File Generator Design Spec

## Overview

Grapes is a CLI tool that generates shell rc files from composable fragments (`.grape` files). Users write shell-agnostic configuration with optional shell-specific overrides, declare dependencies between fragments, and the tool generates properly-ordered rc files for multiple shells.

**Core value:** Define once, generate for bash and zsh. Phase 1 scope.

## File Format

There are two file types:

- **`.grape`** — fragment files containing shell configuration with optional dependencies
- **`.grapes`** — master files that list which fragments to include (entry point for generation)

Both share the same YAML frontmatter + C-style preprocessor body format. They may diverge in the future.

### Frontmatter

```yaml
---
deps: []          # list of fragment names this fragment depends on
phase: main       # "env" or "main" (default: "main")
imports: []       # master rcf only — fragments to include
---
```

- `deps`: array of strings — fragment names (filename without `.grape` extension) that must be processed before this one
- `phase`: determines which output file the fragment contributes to:
  - `env` → `{shell}env` (e.g., `bashenv`, `zshenv`)
  - `main` → `{shell}rc` (e.g., `bashrc`, `zshrc`)
- `imports`: only meaningful in the master `.grapes` file — the list of fragments to include in generation

### Body

Shell content with C-style preprocessor directives:

- `#ifdef BASH` / `#ifdef ZSH` — include only for that shell
- `#ifndef BASH` / `#ifndef ZSH` — exclude for that shell
- `#elif` — else-if branch
- `#else` — else branch
- `#endif` — close conditional block

Content outside directives is shell-agnostic and emitted for all target shells.

### Example: `path.grape`

```yaml
---
deps: []
phase: env
---
export PATH="$HOME/bin:$HOME/.local/bin:$PATH"

#ifdef BASH
export BASH_COMPLETION_DIR="/etc/bash_completion.d"
#endif

#ifdef ZSH
fpath=(/usr/local/share/zsh-completions $fpath)
#endif
```

### Example: `completions.grape`

```yaml
---
deps:
  - path
phase: main
---
#ifdef BASH
source /etc/bash_completion
#endif

#ifdef ZSH
autoload -Uz compinit && compinit
#endif
```

### Example: `master.grapes`

```yaml
---
imports:
  - path
  - prompt
  - aliases
  - completions
---
```

## Architecture

```
master.grapes ──▶ Parser (per-file) ──▶ Dependency Resolver (topo sort)
                                              │
                                     ordered fragments
                                              │
                                              ▼
                                     Preprocessor (per-shell)
                                              │
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                         ~/.config/grapes/
                         ├── bashenv    ├── bashrc    ├── zshenv
                         └── zshrc
```

### Components

1. **Parser** — reads `.grape` and `.grapes` files, splits YAML frontmatter from body, parses frontmatter into a struct, returns raw body with directive markers
2. **Dependency Resolver** — builds a directed graph from fragment `deps`, topologically sorts using Kahn's algorithm, detects cycles and reports with the cycle path
3. **Preprocessor** — walks sorted fragments, evaluates `#ifdef/#ifndef/#elif/#else/#endif` directives per target shell, emits resolved shell content
4. **File Writer** — groups output by phase and shell, writes to `~/.config/grapes/`
5. **Lazy Installer** — (with `--lazy`) appends/removes source marker blocks in user's system rc files

### Dependency Resolution

- Dependencies are transitive (A depends on B, B depends on C → C before B before A)
- Fragments with no deps maintain stable insertion order
- Cycles produce an error with the full cycle path: `circular dependency: a -> b -> a`
- Fragments not reachable from the master's `imports` are not processed

## CLI Interface

### Basic usage

```
grapes <source.grapes>
```

Reads the master `.grapes`, parses all imported fragments, resolves dependencies, preprocesses for bash and zsh, writes output to `~/.config/grapes/`.

### With `--lazy`

```
grapes <source.grapes> --lazy
```

Everything above, plus appends source lines to system rc files using marker blocks:

```bash
# >>> grapes >>>
source "$HOME/.config/grapes/bashrc"
# <<< grapes <<<
```

Source targets:
- `~/.bashenv` or `~/.bash_profile` ← `bashenv` (auto-detect: prefer `~/.bash_profile` if it exists, otherwise `~/.bashenv`)
- `~/.bashrc` ← `bashrc`
- `~/.zshenv` ← `zshenv`
- `~/.zshrc` ← `zshrc`

The marker block allows subsequent runs to update or remove source lines cleanly.

### Repository layout

The CLI entry point lives under `cmd/grapes/main.go`, following the common Go convention of keeping executable entry points in `cmd/` and reusable packages at the module root.

## Output Structure

```
~/.config/grapes/
├── bashenv    # fragments with phase: env, preprocessed for bash
├── bashrc     # fragments with phase: main, preprocessed for bash
├── zshenv     # fragments with phase: env, preprocessed for zsh
└── zshrc      # fragments with phase: main, preprocessed for zsh
```

## Error Handling

- Missing fragment file: error with the expected path
- Circular dependency: error with full cycle path
- Unknown directive: error with file and line number
- Invalid YAML frontmatter: error with file and parse details
- Unknown phase value: error (valid: `env`, `main`)

## Technology

- Language: Go (matches existing project)
- YAML parsing: `gopkg.in/yaml.v3`
- CLI: standard library `flag` or `os.Args` (minimal CLI, no need for cobra for v1)
- No external dependencies beyond YAML parser
