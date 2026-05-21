# grapes

<img width="160" height="160" alt="grapes" src="https://github.com/user-attachments/assets/c5f3908b-d12e-431a-8432-1746f968d411" />

## Overview

`grapes` is a project for managing terminal environments in a more structured way. It is designed to make shell configuration easier to organize, easier to migrate between machines, and easier to keep consistent across multiple shells.

## What does it offer

- A structured approach to terminal environment management
- A shared configuration model for multiple shells
- Modular configuration fragments for tools, languages, and shell setup
- Shell-specific generated output from a single source of configuration intent
- Better portability when moving to a new machine or environment
- Reduced duplication across Bash, Zsh, pwsh, and Nushell
- A clearer separation between environment definition and shell-specific config details

## Repository examples

The repository keeps its example `.grape` and `.grapes` files in `docs/grapes`.

`grapes` now resolves imports from the same directory as the input `.grapes` file only. It does not ship embedded built-in fragments, so example fragments in `docs/grapes` are documentation and local examples rather than runtime defaults baked into the binary.

## Generated files and link targets by shell

Generated shell files are written to the managed output directory:

- Unix-like systems: `~/.config/grapes`
- Windows: `%APPDATA%\grapes`

When linking is enabled, `grapes` adds a managed marker block to the shell's native startup file(s) so those generated files are sourced.

| Shell | Generated files | Link target(s) |
| --- | --- | --- |
| Bash | `bashenv`, `bashrc` | `bashenv` is sourced from `~/.bash_profile` if that file already exists, otherwise from `~/.bashenv`; `bashrc` is sourced from `~/.bashrc` |
| Zsh | `zshenv`, `zshrc` | `zshenv` is sourced from `~/.zshenv`; `zshrc` is sourced from `~/.zshrc` |
| PowerShell (`powershell`, `pwsh`) | `powershell-env.ps1`, `powershell-profile.ps1` | Both files are dot-sourced from a single profile: on Unix-like systems `~/.config/powershell/Microsoft.PowerShell_profile.ps1`; on Windows `~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1` |
| Nushell (`nushell`, `nu`) | `nushell-env.nu`, `nushell-config.nu` | `nushell-env.nu` is `source-env`'d from `~/.config/nushell/env.nu` on Unix-like systems or `%APPDATA%\nushell\env.nu` on Windows; `nushell-config.nu` is sourced from `~/.config/nushell/config.nu` on Unix-like systems or `%APPDATA%\nushell\config.nu` on Windows |

## Disclaimer

This project is still under active development and is not yet ready for general use.
