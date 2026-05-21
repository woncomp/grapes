package renderer

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

func RenderBlock(goos, shell string, env map[string]string, paths []string, body string) (string, error) {
	switch shell {
	case "pwsh", "nushell", "zsh", "bash":
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}

	var lines []string

	for _, key := range slices.Sorted(maps.Keys(env)) {
		switch shell {
		case "pwsh":
			lines = append(lines, fmt.Sprintf("$env:%s = %s", key, QuoteValue(shell, env[key])))
		case "nushell":
			lines = append(lines, fmt.Sprintf("$env.%s = %s", key, QuoteValue(shell, env[key])))
		case "zsh", "bash":
			lines = append(lines, fmt.Sprintf("export %s=%s", key, QuoteValue(shell, env[key])))
		}
	}

	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		switch shell {
		case "pwsh":
			lines = append(lines, fmt.Sprintf(
				"$env:PATH = %s + %s + $env:PATH",
				QuoteValue(shell, path),
				QuoteValue(shell, pwshPathSeparator(goos)),
			))
		case "nushell":
			lines = append(lines, fmt.Sprintf("$env.PATH = ($env.PATH | prepend %s)", QuoteValue(shell, path)))
		case "zsh", "bash":
			lines = append(lines, fmt.Sprintf("export PATH=%s:$PATH", QuoteValue(shell, path)))
		}
	}

	if len(lines) == 0 {
		return body, nil
	}
	if body == "" {
		return strings.Join(lines, "\n") + "\n", nil
	}
	return strings.Join(lines, "\n") + "\n" + body, nil
}

func RenderGrapeExecScope(shell, execPath, execDir, execVersion string) (string, error) {
	switch shell {
	case "pwsh":
		lines := []string{
			fmt.Sprintf("$env:GRAPES_EXEC_PATH = %s", QuoteValue(shell, execPath)),
			fmt.Sprintf("$env:GRAPES_EXEC_DIR = %s", QuoteValue(shell, execDir)),
		}
		if strings.TrimSpace(execVersion) == "" {
			lines = append(lines, "Remove-Item Env:GRAPES_EXEC_VERSION -ErrorAction SilentlyContinue")
		} else {
			lines = append(lines, fmt.Sprintf("$env:GRAPES_EXEC_VERSION = %s", QuoteValue(shell, execVersion)))
		}
		return strings.Join(lines, "\n") + "\n", nil
	case "nushell":
		lines := []string{
			fmt.Sprintf("$env.GRAPES_EXEC_PATH = %s", QuoteValue(shell, execPath)),
			fmt.Sprintf("$env.GRAPES_EXEC_DIR = %s", QuoteValue(shell, execDir)),
		}
		if strings.TrimSpace(execVersion) == "" {
			lines = append(lines, "hide-env GRAPES_EXEC_VERSION")
		} else {
			lines = append(lines, fmt.Sprintf("$env.GRAPES_EXEC_VERSION = %s", QuoteValue(shell, execVersion)))
		}
		return strings.Join(lines, "\n") + "\n", nil
	case "zsh", "bash":
		lines := []string{
			fmt.Sprintf("export GRAPES_EXEC_PATH=%s", QuoteValue(shell, strings.ReplaceAll(execPath, `\`, "/"))),
			fmt.Sprintf("export GRAPES_EXEC_DIR=%s", QuoteValue(shell, strings.ReplaceAll(execDir, `\`, "/"))),
		}
		if strings.TrimSpace(execVersion) == "" {
			lines = append(lines, "unset GRAPES_EXEC_VERSION")
		} else {
			lines = append(lines, fmt.Sprintf("export GRAPES_EXEC_VERSION=%s", QuoteValue(shell, execVersion)))
		}
		return strings.Join(lines, "\n") + "\n", nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func RenderGrapeExecCleanup(shell string) (string, error) {
	switch shell {
	case "pwsh":
		return "Remove-Item Env:GRAPES_EXEC_PATH -ErrorAction SilentlyContinue\nRemove-Item Env:GRAPES_EXEC_DIR -ErrorAction SilentlyContinue\nRemove-Item Env:GRAPES_EXEC_VERSION -ErrorAction SilentlyContinue\n", nil
	case "nushell":
		return "hide-env GRAPES_EXEC_PATH\nhide-env GRAPES_EXEC_DIR\nhide-env GRAPES_EXEC_VERSION\n", nil
	case "zsh", "bash":
		return "unset GRAPES_EXEC_PATH GRAPES_EXEC_DIR GRAPES_EXEC_VERSION\n", nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func pwshPathSeparator(goos string) string {
	if goos == "windows" {
		return ";"
	}
	return ":"
}

func QuoteValue(shell, value string) string {
	switch shell {
	case "pwsh", "nushell":
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case "zsh", "bash":
		return quotePosixDoubleQuotedValue(value)
	default:
		panic(fmt.Sprintf("unsupported shell %q", shell))
	}
}

func quotePosixDoubleQuotedValue(value string) string {
	var escaped strings.Builder
	escaped.Grow(len(value) + 2)
	escaped.WriteByte('"')

	for i := 0; i < len(value); {
		if value[i] != '\\' {
			if value[i] == '"' {
				escaped.WriteString(`\"`)
			} else if value[i] == '`' {
				escaped.WriteString("\\`")
			} else {
				escaped.WriteByte(value[i])
			}
			i++
			continue
		}

		j := i
		for j < len(value) && value[j] == '\\' {
			j++
		}

		count := j - i
		switch {
		case j == len(value):
			escaped.WriteString(strings.Repeat(`\`, count*2))
		case value[j] == '\n':
			escaped.WriteString(strings.Repeat(`\`, count*2))
		case value[j] == '"' || value[j] == '`':
			escaped.WriteString(strings.Repeat(`\`, count*2+1))
			escaped.WriteByte(value[j])
			j++
		default:
			escaped.WriteString(strings.Repeat(`\`, count))
		}

		i = j
	}

	escaped.WriteByte('"')
	return escaped.String()
}
