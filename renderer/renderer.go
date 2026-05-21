package renderer

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

func RenderBlock(goos, shell string, env map[string]string, paths []string, body string) (string, error) {
	switch shell {
	case "bash", "zsh", "nushell", "pwsh":
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}

	var lines []string

	for _, key := range slices.Sorted(maps.Keys(env)) {
		switch shell {
		case "bash", "zsh":
			lines = append(lines, fmt.Sprintf("export %s=%s", key, QuoteValue(shell, env[key])))
		case "nushell":
			lines = append(lines, fmt.Sprintf("$env.%s = %s", key, QuoteValue(shell, env[key])))
		case "pwsh":
			lines = append(lines, fmt.Sprintf("$env:%s = %s", key, QuoteValue(shell, env[key])))
		}
	}

	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		switch shell {
		case "bash", "zsh":
			lines = append(lines, fmt.Sprintf("export PATH=%s:$PATH", QuoteValue(shell, path)))
		case "nushell":
			lines = append(lines, fmt.Sprintf("$env.PATH = ($env.PATH | prepend %s)", QuoteValue(shell, path)))
		case "pwsh":
			lines = append(lines, fmt.Sprintf(
				"$env:PATH = %s + %s + $env:PATH",
				QuoteValue(shell, path),
				QuoteValue(shell, pwshPathSeparator(goos)),
			))
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

func pwshPathSeparator(goos string) string {
	if goos == "windows" {
		return ";"
	}
	return ":"
}

func QuoteValue(shell, value string) string {
	switch shell {
	case "bash", "zsh":
		return quotePosixDoubleQuotedValue(value)
	case "nushell", "pwsh":
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
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
