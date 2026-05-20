# Grapes: Shell-Native Env and Path Rendering Design

## Overview

The current Grapes pipeline expands frontmatter `env` and `paths` fields in the parser using POSIX shell syntax:

- `export KEY="VALUE"`
- `export PATH="ENTRY:$PATH"`

That works for `bash` and `zsh`, but it breaks the newly added `nushell` and `pwsh` targets because their managed files now contain shell-native startup-file locations and load commands, but still receive POSIX-flavored generated content.

This design extends Grapes so `env` and `paths` render natively for every supported shell while preserving the existing fragment model and phase model.

## Goals

- Keep `.grape` and `.grapes` frontmatter unchanged for users.
- Preserve the existing `env` and `main` phase model.
- Render `env` and `paths` using shell-native syntax for:
  - `bash`
  - `zsh`
  - `nushell`
  - `pwsh`
- Keep parsing shell-agnostic.
- Close the runtime gap where `pwsh` and `nushell` files can be linked but still contain incompatible generated content.

## Non-Goals

- Change the frontmatter schema.
- Redesign preprocessing directives.
- Add more shell phases.
- Change existing fragment behavior when `env` and `paths` are not used.

## Current Problem

Today `parser.expandBlock()` converts structured frontmatter into concrete shell code too early. At parse time Grapes does not yet know which target shell is being rendered, so the parser bakes in one syntax shape for all shells.

That means:

- `bash` and `zsh` are correct
- `nushell` receives POSIX `export ...` lines
- `pwsh` receives POSIX `export ...` lines

The shell-registration work is therefore incomplete at runtime for fragments that use frontmatter-driven environment or path setup.

## Recommended Approach

Move shell-specific `env` and `paths` rendering out of the parser and into a shell-aware rendering step that runs after parsing, when the target shell is known.

This keeps responsibilities clean:

- parser: read fragment structure and preserve frontmatter data
- renderer: translate frontmatter data into target-shell text
- preprocessor: keep handling shell-condition directives
- writer: keep writing assembled managed files

## Architecture

### Parser responsibilities

The parser should preserve structured block data instead of flattening `env` and `paths` into POSIX shell text.

Each parsed block should carry:

- `Phase`
- `Env`
- `Paths`
- `Body`

If the current parsed block type stores only rendered `Body`, this extension should add explicit `Env map[string]string` and `Paths []string` fields to parsed blocks so downstream rendering can operate on structured data.

### Rendering responsibilities

Introduce a small shell-aware rendering step that accepts:

- canonical shell name
- block `env`
- block `paths`
- raw body

and returns the shell-native text for that target.

Conceptually:

```go
func renderBlock(shell string, env map[string]string, paths []string, body string) string
```

This function should:

1. render environment assignments in deterministic key order
2. render path prepend operations in declared order
3. append the raw block body unchanged

## Data Flow

The current `run()` flow already iterates by target shell and block phase. That is the right point to render shell-native text because the shell is known there.

Updated flow:

```text
parse fragment -> keep structured block data
resolve fragments
for each selected shell
  for each block
    render env/paths/body for that shell
    preprocess rendered text for that shell
write managed files
```

This keeps the feature local to the orchestration/rendering boundary and avoids threading shell identity back into parsing.

## Shell Syntax Rules

### Bash and Zsh

Keep the current output shape:

```sh
export KEY="VALUE"
export PATH="ENTRY:$PATH"
```

### Nushell

Render frontmatter using Nu-native syntax.

Environment variables:

```nu
$env.NAME = "value"
```

Path prepend:

```nu
$env.PATH = ($env.PATH | prepend "entry")
```

This exact prepend form should be used so the spec is testable and unambiguous.

### pwsh

Render frontmatter using pwsh-native syntax.

Environment variables:

```pwsh
$env:NAME = "value"
```

Path prepend:

```pwsh
$env:PATH = "entry;$env:PATH"
```

## Compatibility

- Fragments that do not use `env` or `paths` should be unaffected.
- Existing `bash` and `zsh` output should remain unchanged.
- Existing preprocessor behavior should remain unchanged.
- The new rendering step should work with current `#ifdef` / `#elif` processing because rendering happens before preprocessing for the active shell.

## Error Handling

- Unsupported shells should still fail through the shell-selection layer, not through the renderer.
- Empty `env` and `paths` should produce no extra lines.
- Rendering should remain deterministic so output and tests stay stable.

## Testing Strategy

Add focused tests for the rendering behavior and end-to-end regression coverage.

### Unit-level coverage

Add renderer tests that verify shell-native `env` and `paths` output for:

- `bash`
- `zsh`
- `nushell`
- `pwsh`

These tests should cover:

- multiple env keys sorted deterministically
- path prepend order preservation
- body passthrough after generated lines

### End-to-end coverage

Add CLI or writer-facing tests that generate managed files for `nushell` and `pwsh` from fragments using frontmatter `env` and `paths`, then assert:

- output files exist
- output contains shell-native syntax
- output does not contain POSIX `export ...` syntax for the new shells

## Implementation Boundaries

To keep this extension focused:

1. make the smallest parser change needed to preserve structured frontmatter data
2. add one shell-aware rendering layer rather than scattering syntax branches through the pipeline
3. keep shell syntax choices localized and testable
4. avoid unrelated refactoring

## Validation

This extension is complete when:

- `go test ./...` passes
- frontmatter `env` and `paths` render correctly for all supported shells
- `nushell` and `pwsh` managed files no longer contain incompatible POSIX exports for those frontmatter-driven lines

## Summary

The right fix is to delay `env` and `paths` rendering until Grapes knows the target shell. That keeps parsing shell-agnostic, preserves the current fragment model, and makes the newly added `nushell` and `pwsh` targets functionally complete for frontmatter-driven environment and path setup.
