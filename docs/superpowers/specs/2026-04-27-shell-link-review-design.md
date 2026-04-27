# Grapes: Shell Link Review Before RC/Profile Changes

## Overview

Today, `grapes` writes link snippets directly into user-owned shell rc/profile files during the link step. This design changes that flow so linking becomes review-first by default: before `grapes` modifies any user-owned rc/profile file, it shows the pending changes and requires explicit confirmation.

The review is grouped by target shell rather than by file. For each selected shell, `grapes` presents a single dialog that includes diffs for all user-owned rc/profile files that shell would modify. Approving applies that shell's pending link changes. Rejecting skips that shell and continues to the next selected shell.

This keeps the existing parse -> resolve -> preprocess -> write -> link pipeline intact while adding a clear safety checkpoint around the highest-risk file modifications.

## Goals

- Show the exact pending changes before modifying any user-owned rc/profile file.
- Make review-and-confirmation the default behavior for shell linking.
- Group confirmation by selected shell instead of prompting once per file.
- Keep `--nolink` behavior unchanged.
- Support unattended runs with an explicit `--yes` bypass.
- Avoid prompting or rewriting when a shell's link targets are already up to date.
- Render diffs with ANSI colors when stdout is a terminal/TTY, while keeping non-TTY output plain text.

## Non-Goals

- Redesign how managed Grapes output files are generated.
- Add backups, rollback, or transactional writes across multiple shells.
- Include Grapes-managed output files in the review dialog.
- Introduce a general-purpose diff framework for unrelated features.

## Current Flow

The current link flow in `cmd/grapes/main.go`:

1. generates managed files under the Grapes output directory
2. resolves selected shell targets
3. asks each shell for its `LinkTargets`
4. calls `shells.Install` for each target file

`shells.Install` already handles the write semantics that matter here:

- read existing file content
- remove or replace an existing Grapes marker block
- preserve user content outside the marker block
- create parent directories if needed
- write the updated rc/profile file

That behavior should remain centralized.

## Recommended Approach

Add a review stage between link-target discovery and `shells.Install`.

For each selected shell:

1. collect that shell's user-owned link targets
2. compute the proposed post-install contents for each target without writing
3. discard unchanged targets from the review set
4. if no targets would change, report that no rc/profile changes are needed for that shell
5. otherwise show a single grouped review for that shell
6. if approved, apply `shells.Install` to each changed target for that shell
7. if rejected, skip that shell and continue to the next selected shell

This design matches the user's preference for one shell-level confirmation dialog while keeping file-level write behavior simple and testable.

## CLI Contract

### Default behavior

Linking becomes confirmation-first by default.

If `grapes` is about to modify one or more user-owned link target files for a selected shell, it must:

1. show the diff review for that shell
2. prompt for confirmation
3. apply the changes only after approval

### `--yes`

Add:

- `-y`, `--yes`: approve all shell-link review prompts and proceed with linking

`--yes` is the explicit bypass for CI, scripting, and other unattended runs.

It does not disable no-op detection. Unchanged link targets must still be skipped without rewriting.

### `--nolink`

`--nolink` continues to skip shell linking entirely. When it is set, no review dialog is shown because no user-owned rc/profile files are modified.

## Review Model

The review unit is a selected shell, not an individual file.

Examples:

- `bash` may review one or more user-owned link targets together
- `zsh` may review one or more user-owned link targets together
- `powershell` may review its native profile target as one shell-level unit

Each shell review contains only that shell's user-owned rc/profile link targets. Grapes-managed output files under the managed output directory are out of scope for the dialog.

### Review contents

For each changed target in the shell review:

- identify the file path being reviewed
- show a readable diff from current contents to proposed contents

After all diffs for that shell are shown, prompt once. When stdout is a terminal/TTY, the diff output must be colorized so additions, removals, and headers are easy to distinguish. When stdout is not a terminal/TTY, the same diff must be emitted without ANSI color codes.

Prompt once:

```text
Apply these shell link changes? [y/N]:
```

Approval applies all changed link targets for that shell. Rejection skips all changed link targets for that shell.

## Architecture

### Keep writes centralized in `shells.Install`

The actual file write path should remain in `shells.Install`. This preserves current behavior and avoids duplicating marker-block update logic in the CLI layer.

### Add preflight helpers in `shells`

The `shells` package should expose helper logic for the review step that can:

- read current rc/profile contents
- compute the exact contents that `Install` would produce
- determine whether the target is unchanged
- render a readable diff for display

One practical shape is to add a helper that returns a per-target review description, for example:

```go
type InstallPreview struct {
    RCFile           string
    CurrentContent   string
    ProposedContent  string
    Changed          bool
    Diff             string
}
```

The exact helper API can vary, but it should keep the install rules in one place and make review logic reusable in tests.

### Orchestration in `cmd/grapes/main.go`

The CLI layer should:

1. gather each selected shell's link targets
2. compute previews for that shell
3. short-circuit if none changed
4. present grouped diffs
5. prompt unless `--yes` is set
6. call `shells.Install` for each changed target in that shell only after approval

This keeps shell grouping in the orchestration layer, where the selected target shell is already known.

## Diff Format

The output does not need a third-party diff dependency. A simple unified-style textual diff is sufficient if it is clear and deterministic.

Minimum requirements:

- file header identifying the rc/profile file
- added lines prefixed with `+`
- removed lines prefixed with `-`
- support for both new-file creation and updates to an existing Grapes marker block
- ANSI-colored terminal output when stdout is a TTY

Color behavior:

- additions should be rendered in green
- removals should be rendered in red
- diff headers and hunk markers should use a distinct neutral or accent color
- color must be disabled automatically when stdout is not a terminal/TTY

For a missing target file, the current content is treated as empty and the diff shows the full inserted content as additions.

## Error Handling

### No-op targets

If a shell's proposed link targets are all unchanged:

- do not prompt
- do not rewrite those files
- report that no rc/profile changes are needed for that shell

### Rejection

If the user rejects a shell review:

- skip all changed user-owned link targets for that shell
- continue reviewing later selected shells
- report that the shell was skipped

### Non-interactive input

If a shell needs confirmation, `--yes` is not set, and confirmation input is unavailable or unreadable, the command must fail with a clear error.

The error should tell the user to:

- rerun with `--yes` for unattended approval, or
- rerun with `--nolink` to skip user rc/profile linking

The command must not silently assume approval.

### Partial application across shells

This design does not make the full run atomic across several shells. A previously approved shell may already be linked before a later shell is rejected or errors.

## Testing Strategy

Add focused coverage for both preview behavior and CLI orchestration.

### Unit tests

Add tests in `shells/shells_test.go` for:

- computing proposed contents for a missing rc/profile file
- computing proposed contents when replacing an existing Grapes marker block
- unchanged detection when the installed result already matches the file
- diff rendering for new-file and update scenarios
- colorized vs plain diff output behavior based on whether stdout is a TTY
- confirmation parsing for `y` / `yes`
- rejection parsing for empty input, `n`, and `no`
- unreadable or EOF confirmation input

### Integration tests

Add tests in `cmd/grapes/main_test.go` for:

- grouped shell review showing one confirmation decision per shell
- approving a shell applies all changed link targets for that shell
- rejecting a shell skips that shell and continues to later shells
- unchanged targets do not prompt or rewrite
- `--yes` bypasses prompts and still skips unchanged targets
- non-interactive runs without `--yes` fail clearly when link changes are pending
- `--nolink` still bypasses the review path

## Validation

This design is complete when:

- running `grapes` with linking enabled and pending shell link changes shows grouped per-shell diffs before writes
- diff output is colorized on TTYs and plain on non-TTY output
- approval applies only the reviewed shell's changed link targets
- rejection skips only the reviewed shell and allows the run to continue
- unchanged shell link targets are not rewritten and do not prompt
- `--yes` supports unattended runs
- non-interactive runs without `--yes` do not silently modify user-owned rc/profile files

## Summary

The right change is to make rc/profile linking review-first by default, grouped by selected shell. The CLI should compute previews before writing, show all user-owned link-target diffs for one shell together, prompt once, and then either install that shell's changes or skip them as a unit. The actual file write semantics remain centralized in `shells.Install`, which keeps the feature focused, testable, and consistent with the current architecture.
