package preprocessor

import (
	"fmt"
	"strings"

	"github.com/woncomp/grapes/renderer"
)

// Process evaluates preprocessor directives in body for the given shell.
// Supported directives: #ifdef, #ifndef, #elif, #else, #endif.
func Process(body string, shell string) (string, error) {
	trimmedBody := strings.TrimRight(body, "\n")
	var lines []string
	if trimmedBody != "" {
		lines = strings.Split(trimmedBody, "\n")
	}
	output := make([]string, 0, len(lines))
	stack := []blockState{{include: true, satisfied: true}}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if isDirective(trimmed) {
			err := handleDirective(trimmed, shell, &stack, i+1)
			if err != nil {
				return "", err
			}
			continue
		}

		// Detect unknown preprocessor-like directives
		if strings.HasPrefix(trimmed, "#") && trimmed != "" {
			return "", fmt.Errorf("line %d: unknown directive %q", i+1, trimmed)
		}

		if currentInclude(stack) {
			output = append(output, line)
		}
	}

	if len(stack) != 1 {
		return "", fmt.Errorf("unterminated directive (unclosed #ifdef/#ifndef)")
	}

	// Join and add trailing newline
	if len(output) == 0 {
		return "", nil
	}
	return strings.Join(output, "\n") + "\n", nil
}

func ShellInjectionLine(shell string) string {
	switch strings.ToLower(shell) {
	case "nushell":
		return fmt.Sprintf(`$env.GRAPES_SHELL = "%s"`, shell)
	case "pwsh":
		return fmt.Sprintf(`$env:GRAPES_SHELL = "%s"`, shell)
	default:
		return fmt.Sprintf(`export GRAPES_SHELL="%s"`, shell)
	}
}

func OutputPathInjectionLine(shell string, outputPath string) string {
	formattedPath := outputPath
	switch strings.ToLower(shell) {
	case "bash", "zsh":
		formattedPath = strings.ReplaceAll(outputPath, `\`, "/")
		return fmt.Sprintf("export GRAPES_OUTPUT_PATH=%s", renderer.QuoteValue(shell, formattedPath))
	case "nushell":
		return fmt.Sprintf("$env.GRAPES_OUTPUT_PATH = %s", renderer.QuoteValue(shell, formattedPath))
	case "pwsh":
		return fmt.Sprintf("$env:GRAPES_OUTPUT_PATH = %s", renderer.QuoteValue(shell, formattedPath))
	default:
		panic(fmt.Sprintf("unsupported shell %q", shell))
	}
}

func InjectedEnvLines(shell string, outputPath string) []string {
	return []string{
		ShellInjectionLine(shell),
		OutputPathInjectionLine(shell, outputPath),
	}
}

type blockState struct {
	include   bool // whether content in this block should be included
	satisfied bool // whether any branch has already matched
}

func currentInclude(stack []blockState) bool {
	for _, s := range stack {
		if !s.include {
			return false
		}
	}
	return true
}

func isDirective(line string) bool {
	return strings.HasPrefix(line, "#ifdef ") ||
		strings.HasPrefix(line, "#ifndef ") ||
		strings.HasPrefix(line, "#elif ") ||
		line == "#else" ||
		line == "#endif"
}

func handleDirective(line string, shell string, stack *[]blockState, lineNum int) error {
	parts := strings.Fields(line)
	directive := parts[0]

	switch directive {
	case "#ifdef":
		if len(parts) != 2 {
			return fmt.Errorf("line %d: #ifdef requires exactly one argument", lineNum)
		}
		match := strings.EqualFold(parts[1], shell)
		parentInclude := currentInclude(*stack)
		*stack = append(*stack, blockState{
			include:   parentInclude && match,
			satisfied: match,
		})

	case "#ifndef":
		if len(parts) != 2 {
			return fmt.Errorf("line %d: #ifndef requires exactly one argument", lineNum)
		}
		match := !strings.EqualFold(parts[1], shell)
		parentInclude := currentInclude(*stack)
		*stack = append(*stack, blockState{
			include:   parentInclude && match,
			satisfied: match,
		})

	case "#elif":
		if len(*stack) < 2 {
			return fmt.Errorf("line %d: #elif without matching #ifdef/#ifndef", lineNum)
		}
		if len(parts) != 2 {
			return fmt.Errorf("line %d: #elif requires exactly one argument", lineNum)
		}
		top := &(*stack)[len(*stack)-1]
		if top.satisfied {
			top.include = false
		} else {
			match := strings.EqualFold(parts[1], shell)
			parentInclude := true
			if len(*stack) > 1 {
				parentInclude = (*stack)[len(*stack)-2].include
			}
			top.include = parentInclude && match
			top.satisfied = top.satisfied || match
		}

	case "#else":
		if len(*stack) < 2 {
			return fmt.Errorf("line %d: #else without matching #ifdef/#ifndef", lineNum)
		}
		top := &(*stack)[len(*stack)-1]
		if top.satisfied {
			top.include = false
		} else {
			parentInclude := true
			if len(*stack) > 1 {
				parentInclude = (*stack)[len(*stack)-2].include
			}
			top.include = parentInclude
			top.satisfied = true
		}

	case "#endif":
		if len(*stack) < 2 {
			return fmt.Errorf("line %d: #endif without matching #ifdef/#ifndef", lineNum)
		}
		*stack = (*stack)[:len(*stack)-1]

	default:
		return fmt.Errorf("line %d: unknown directive %q", lineNum, directive)
	}

	return nil
}
