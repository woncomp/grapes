# Windows Executable Dependency Compatibility Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Make `depend_executable` behave reliably on Windows when tools are installed outside PATH but in common Windows shim/bin locations or when fallback lookup must resolve `.exe`/PATHEXT-style filenames.

**Architecture:** Keep `depend_executable` as the executable-oriented dependency model, but make its runtime lookup OS-aware. Add a `goos`-aware dependency checker configuration, split common search path logic by platform, teach fallback lookup to try Windows executable suffixes, improve home-directory expansion for Windows environments that use `USERPROFILE`, and lock the behavior down with targeted unit tests.

**Tech Stack:** Go 1.26, standard library only, existing `cmd/grapes` dependency checker and Go package tests

---

## Windows incompatibilities to address

- `commonExecutableSearchPaths()` is Unix-only and ignores Windows package manager/shim directories.
- Fallback lookup appends the raw binary name only, so `search_paths` and common-path scanning miss `tool.exe` when config says `binary: tool`.
- `~` expansion only consults `HOME`, which is often absent or unreliable on Windows compared with `USERPROFILE`.
- There is no test coverage proving Windows fallback path scanning, PATHEXT-style suffix resolution, or Windows-specific environment expansion.

## File Structure

### Existing files to modify

- `cmd/grapes/dependencies.go`
- `cmd/grapes/dependencies_test.go`
- `docs/plans/2026-04-27-windows-executable-dependency-compatibility.md`

### New files to create

- None

## Task 1: Add failing Windows lookup tests first

- [ ] Write a failing test in `cmd/grapes/dependencies_test.go` that proves common Windows shim/bin directories are searched when `exec.LookPath` fails.
- [ ] Write a failing test that proves fallback lookup resolves `.exe` when `binary: bun` is configured and only `bun.exe` exists in a scanned directory.
- [ ] Write a failing test that proves `~` expansion falls back to `USERPROFILE` when `HOME` is missing.
- [ ] Run `go test ./cmd/grapes -run 'TestExecutableDependencyCheckWindows|TestExpandSearchPathsUsesUserProfile'` and confirm the tests fail for the intended missing-behavior reasons.

## Task 2: Make executable scanning OS-aware

- [ ] Add `goos` to `dependencyCheckOptions`, defaulting to `runtime.GOOS` inside `checkGrapeDependencies`.
- [ ] Refactor the common executable search path helper to branch by OS.
- [ ] For Windows, include the most valuable high-signal fallback directories such as `%LOCALAPPDATA%\Microsoft\WinGet\Links`, `%USERPROFILE%\scoop\shims`, `%ChocolateyInstall%\bin`, and `%APPDATA%\npm` when the corresponding environment variables exist.
- [ ] Keep existing Unix behavior unchanged for non-Windows platforms.

## Task 3: Teach fallback lookup to resolve Windows executable suffixes

- [ ] Add a helper that expands candidate executable names for direct path scanning.
- [ ] On non-Windows, keep the existing single-name behavior.
- [ ] On Windows, if the configured binary name has no extension, probe PATHEXT-style executable suffixes (at minimum `.exe`, `.cmd`, `.bat`, `.com`) in addition to the raw name.
- [ ] Reuse the expanded candidate names for both common-path scanning and configured `search_paths` scanning.
- [ ] Ensure the returned `LOCATION` is the actual matched file path, including its extension.

## Task 4: Improve Windows home expansion and verify all tests

- [ ] Add a small helper to resolve the effective home directory for path expansion, preferring `HOME` when set and otherwise using `USERPROFILE` on Windows.
- [ ] Use that helper in `expandSearchPaths()` so `~` works on Windows without requiring `HOME`.
- [ ] Run `go test ./cmd/grapes`.
- [ ] Run `go test ./...`.

## Task 5: Commit, merge, push, and clean up

- [ ] Commit the focused Windows compatibility changes with a message scoped to executable dependency lookup.
- [ ] Merge the worktree branch back into `main` locally.
- [ ] Run `go test ./...` again on merged `main`.
- [ ] Push `main` to `origin`.
- [ ] Remove the worktree and delete the feature branch.

## Review Checklist

- [ ] Windows fallback lookup searches useful common Windows shim/bin directories.
- [ ] Windows fallback lookup can find `tool.exe` when config says `binary: tool`.
- [ ] `search_paths` fallback works on Windows without requiring explicit executable extensions in the config.
- [ ] `~` expansion works on Windows via `USERPROFILE` if `HOME` is absent.
- [ ] Non-Windows executable lookup behavior remains unchanged.
- [ ] Full test suite passes before and after merge/push.
