# Grapes: Nushell and pwsh Support Design

## Overview

Grapes currently targets `bash` and `zsh`. This design adds first-class support for `nushell` and `pwsh` while preserving the existing fragment pipeline and the current two-phase model:

- `env`
- `main`

The goal is to let users generate managed startup files for these shells, link native shell startup files back to Grapes-managed outputs, and use shell-specific preprocessor conditionals without forcing a large rewrite of the current architecture.

## Goals

- Add `nushell` and `pwsh` as supported target shells.
- Keep the existing parse -> resolve -> preprocess -> write -> link flow.
- Preserve the current fragment model and phase semantics.
- Support native startup-file linking on Unix-like systems and Windows.
- Extend shell conditionals so fragments can target the new shells directly.
- Keep shell-specific behavior isolated inside the `shells` package.

## Non-Goals

- Redesign the fragment model beyond the existing `env` and `main` phases.
- Introduce arbitrary numbers of startup phases per shell.
- Add broad shell auto-detection heuristics that guess when the environment is ambiguous.
- Change existing `bash` or `zsh` behavior except where required by the new shell-linking abstraction.

## Current Constraints

Today, Grapes assumes a POSIX-style source command during link installation:

```go
source "<managed-path>"
```

That works for `bash` and `zsh`, but it is not sufficient for the new shells:

- `nushell` needs `source-env` for environment-stage loading and `source` for config-stage loading.
- `pwsh` uses dot-sourcing:

```pwsh
. "<managed-path>"
```

In addition, the native startup-file locations differ by shell and operating system.

## Proposed Architecture

The parser, resolver, preprocessor, and writer remain structurally unchanged. The design focuses on a small upgrade to the `shells` package so that shell integration becomes expressive enough for non-POSIX startup semantics.

### Shell responsibilities

Each shell implementation remains responsible for:

- canonical target name
- aliases
- managed filename per phase
- native rc/profile targets

The key change is that link installation must become shell-specific instead of hardcoding the line:

```text
source "<managed-path>"
```

### Link abstraction upgrade

Replace the current install contract of:

- rc file path
- managed source path

with a richer per-target description that can also render the install snippet. Conceptually, each link target should answer:

- which native startup file is modified
- which managed Grapes file is referenced
- what exact line or block should be inserted into that native file

One practical shape is:

```go
type LinkTarget struct {
    RCFile       string
    InstallLines []string
}
```

`shells.Install` would then install the provided shell-specific lines inside the existing Grapes marker block instead of constructing a POSIX `source` line internally.

This keeps marker-block behavior, idempotent replacement, and uninstall behavior centralized while letting each shell define the correct native load syntax.

## Shell Mapping

The two Grapes phases remain the user-facing contract:

- `env`
- `main`

Each supported shell maps those phases to native startup files and managed filenames.

### Bash

No behavior change:

- `env` -> managed `bashenv`
- `main` -> managed `bashrc`
- link targets:
  - `~/.bash_profile` if present, otherwise `~/.bashenv`
  - `~/.bashrc`
- install line:

```bash
source "<managed-path>"
```

### Zsh

No behavior change:

- `env` -> managed `zshenv`
- `main` -> managed `zshrc`
- link targets:
  - `~/.zshenv`
  - `~/.zshrc`
- install line:

```bash
source "<managed-path>"
```

### Nushell

Nushell keeps separate native startup files for environment setup and general configuration, which matches the current two-phase Grapes model well.

- `env` -> managed `nushell-env.nu`
- `main` -> managed `nushell-config.nu`

Native targets:

- Unix-like systems:
  - `~/.config/nushell/env.nu`
  - `~/.config/nushell/config.nu`
- Windows:
  - `%APPDATA%\nushell\env.nu`
  - `%APPDATA%\nushell\config.nu`

Install lines by phase:

```nu
source-env `<managed-path>`
source `<managed-path>`
```

Mapping:

- native `env.nu` loads Grapes `env` output using `source-env`
- native `config.nu` loads Grapes `main` output using `source`

This preserves the meaning of Grapes phases while respecting Nushell's environment propagation rules.

### pwsh

pwsh does not have separate built-in profile files for "env phase" and "main phase" in the same way that `bash`, `zsh`, and `nushell` do. To preserve Grapes's two-phase model without redesigning fragments, Grapes will generate two managed pwsh files and load both from the user's current-user current-host profile.

- `env` -> managed `pwsh-env.ps1`
- `main` -> managed `pwsh-profile.ps1`

Native target:

- Unix-like systems:
  - `~/.config/powershell/Microsoft.PowerShell_profile.ps1`
- Windows:
  - `%HOME%\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`

Install lines:

```pwsh
. "<managed-path>"
```

Load order inside a single Grapes marker block in the native profile:

1. dot-source Grapes `env` output
2. dot-source Grapes `main` output

That preserves phase ordering while fitting naturally into pwsh's single-profile startup model.

## Managed Output Directory

The current implementation writes managed files to `~/.local/state/grapes` across platforms.

Managed output directory:

- Unix-like systems:
  - `~/.local/state/grapes`
- Windows:
  - `~/.local/state/grapes`

The CLI should use a shared managed-output resolver instead of hardcoding the managed output path.

Home-like path resolution should also become explicit:

- prefer `HOME` where appropriate
- on Windows, allow `USERPROFILE` as the fallback home directory when `HOME` is absent
- use the home directory for the managed output location

This keeps the generated file paths consistent with the native profile locations that will reference them.

## Path Resolution Rules

Shell implementations resolve native target locations using explicit, platform-aware rules.

### Shared rules

- Path resolution should be deterministic and testable.
- Parent directories for native target files must be created before installation if they do not already exist.
- Fail with an explicit error when a required environment variable is missing.

### Nushell path resolution

- On Unix-like systems, derive the config directory from `HOME` as `~/.config/nushell`.
- On Windows, derive the config directory from `APPDATA` as `%APPDATA%\nushell`.
- If the required base environment variable is missing, return an explicit error.

### pwsh path resolution

- On Unix-like systems, use `~/.config/powershell/Microsoft.PowerShell_profile.ps1`.
- On Windows, use `%USERPROFILE%\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`, with `HOME` accepted if it is already set and preferred by the process environment.
- If both `HOME` and `USERPROFILE` are missing where a home-derived path is required, return an explicit error.

This design intentionally uses the documented default current-user current-host profile location instead of attempting to invoke pwsh to discover `$PROFILE` dynamically.

## Preprocessor Changes

The preprocessor already compares directive operands to the active shell name case-insensitively. Adding the new shells mainly requires test and documentation updates.

New supported examples:

- `--#ifdef NUSHELL`
- `--#ifndef NUSHELL`
- `--#elif NUSHELL`
- `--#ifdef PWSH`
- `--#ifndef PWSH`
- `--#elif PWSH`

`GRAPES_SHELL` should continue to expose the canonical shell name:

- `bash`
- `zsh`
- `nushell`
- `pwsh`

This keeps shell gating simple and consistent with the existing design.

## CLI Behavior

### Target parsing

Add the following aliases:

- Nushell:
  - `nushell`
  - `nu`
- pwsh:
  - `pwsh`

Canonical names returned by the shell layer remain:

- `nushell`
- `pwsh`

This ensures:

- stable filenames
- stable `GRAPES_SHELL` values
- deduplication across aliases

### Default target detection

Keep the current conservative behavior:

- If Grapes can reliably detect the shell from environment input, use it.
- If not, fail with the existing explicit guidance to pass `-t`.

This feature does not broaden shell auto-detection into heuristic guessing for Windows terminal environments. Users can always select `-t nushell` or `-t pwsh` explicitly, and that behavior remains fully supported.

## Managed Output Layout

With the new shells included, managed files in the Grapes output directory may now include:

```text
~/.local/state/grapes/
├── bashenv
├── bashrc
├── zshenv
├── zshrc
├── nushell-env.nu
├── nushell-config.nu
├── pwsh-env.ps1
└── pwsh-profile.ps1
```

Pruning behavior remains unchanged in spirit: Grapes should delete managed outputs for supported shells that were not selected in the current run.

## Installation Behavior

The existing Grapes marker block format remains:

```text
# >>> grapes >>>
<install line(s)>
# <<< grapes <<<
```

The installation logic should remain idempotent:

- replace an existing Grapes marker block in place
- preserve unrelated user content
- keep uninstall behavior symmetrical

The only change is that the installed content becomes shell-specific and may contain multiple lines when a single native startup file must load more than one managed Grapes file.

## Error Handling

The implementation should continue to prefer explicit failures over silent fallbacks.

New or clarified error cases:

- unsupported target alias still reports supported targets
- missing `APPDATA` for Windows Grapes output or Nushell profile resolution
- missing both `HOME` and `USERPROFILE` where a home-derived path is required
- inability to create required profile parent directories
- inability to write native shell startup files

Existing error behavior remains for:

- unknown CLI flags
- invalid fragment syntax
- missing dependencies
- circular dependencies
- malformed preprocessor directives

## Testing Strategy

The feature should extend the existing package-level test suite rather than introduce a new test harness.

### Shell package tests

- `SupportedNames()` includes `nushell` and `pwsh`
- alias parsing works for `nu` and `pwsh`
- managed filenames map correctly for both phases
- native target resolution works on Unix-like and Windows inputs
- managed output directory resolution works on Unix-like and Windows inputs
- install marker blocks render the correct load syntax:
  - POSIX `source`
  - Nushell `source-env` and `source`
  - pwsh ordered dot-sourcing inside one marker block
- parent-directory creation behavior is covered

### Preprocessor tests

- `--#ifdef NUSHELL` and `--#ifdef PWSH` behave correctly
- `--#elif NUSHELL` and `--#elif PWSH` behave correctly
- `GRAPES_SHELL` is emitted as `nushell` and `pwsh`

### CLI tests

- explicit targets accept `nu`, `nushell`, and `pwsh`
- duplicate aliases deduplicate to one canonical target
- unsupported targets still fail cleanly

### End-to-end run tests

- `--nolink` writes only the selected new shell outputs
- linking installs the correct marker block into native Nushell files
- linking installs one Grapes marker block with two ordered dot-source lines into the native pwsh profile
- pruning removes stale managed outputs for unselected shells

## Migration and Compatibility

- Existing `bash` and `zsh` behavior remains unchanged.
- Existing `bash` and `zsh` fragments continue to work without modification.
- Fragments may opt into `nushell` or `pwsh` using preprocessor conditionals.
- No changes are required to the parser, resolver, or writer APIs visible to fragment authors.

## Recommended Implementation Boundaries

To keep this feature well-bounded:

1. confine shell-specific path and install-line logic to `shells/`
2. keep the CLI orchestration model intact
3. keep the fragment phase model unchanged
4. avoid introducing generic "infinite shell phases" abstractions

This gives Grapes room to support more shells later without paying the cost of a full architecture redesign now.

## Validation

The implementation should be considered complete when:

- `go test ./...` passes
- the new shell targets generate correct managed files
- native linking is idempotent on supported OS path schemes
- preprocessor conditionals work for `NUSHELL` and `PWSH`

## Summary

The recommended design is a small abstraction upgrade centered on shell-specific link installation. It keeps Grapes's current architecture intact, preserves the two-phase fragment model, adds native startup integration for `nushell` and `pwsh` across Unix-like systems and Windows, and limits change to the places that truly need shell-specific behavior.
