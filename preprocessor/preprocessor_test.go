package preprocessor

import (
	"strings"
	"testing"
)

func shellLine(shell string) string {
	return ""
}

func TestNoDirectives(t *testing.T) {
	input := "export FOO=bar\necho hello\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("bash") + input
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestShellInjection(t *testing.T) {
	tests := map[string]string{
		"bash":    `export GRAPES_SHELL="bash"`,
		"zsh":     `export GRAPES_SHELL="zsh"`,
		"nushell": `$env.GRAPES_SHELL = "nushell"`,
		"pwsh":    `$env:GRAPES_SHELL = "pwsh"`,
	}

	for shell, want := range tests {
		if got := ShellInjectionLine(shell); got != want {
			t.Errorf("ShellInjectionLine(%q) = %q, want %q", shell, got, want)
		}
	}
}

func TestOutputPathInjection(t *testing.T) {
	tests := []struct {
		shell      string
		outputPath string
		want       string
	}{
		{
			shell:      "bash",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `export GRAPES_OUTPUT_PATH="C:/Users/me/AppData/Roaming/grapes"`,
		},
		{
			shell:      "zsh",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `export GRAPES_OUTPUT_PATH="C:/Users/me/AppData/Roaming/grapes"`,
		},
		{
			shell:      "nushell",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `$env.GRAPES_OUTPUT_PATH = 'C:\Users\me\AppData\Roaming\grapes'`,
		},
		{
			shell:      "pwsh",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `$env:GRAPES_OUTPUT_PATH = 'C:\Users\me\AppData\Roaming\grapes'`,
		},
	}

	for _, tt := range tests {
		if got := OutputPathInjectionLine(tt.shell, tt.outputPath); got != tt.want {
			t.Errorf("OutputPathInjectionLine(%q, %q) = %q, want %q", tt.shell, tt.outputPath, got, tt.want)
		}
	}
}

func TestHomeInjection(t *testing.T) {
	tests := []struct {
		shell    string
		homePath string
		want     string
	}{
		{
			shell:    "bash",
			homePath: `C:\Users\me\src\dotfiles`,
			want:     `export GRAPES_HOME="C:/Users/me/src/dotfiles"`,
		},
		{
			shell:    "zsh",
			homePath: `C:\Users\me\src\dotfiles`,
			want:     `export GRAPES_HOME="C:/Users/me/src/dotfiles"`,
		},
		{
			shell:    "nushell",
			homePath: `C:\Users\me\src\dotfiles`,
			want:     `$env.GRAPES_HOME = 'C:\Users\me\src\dotfiles'`,
		},
		{
			shell:    "pwsh",
			homePath: `C:\Users\me\src\dotfiles`,
			want:     `$env:GRAPES_HOME = 'C:\Users\me\src\dotfiles'`,
		},
	}

	for _, tt := range tests {
		if got := HomeInjectionLine(tt.shell, tt.homePath); got != tt.want {
			t.Errorf("HomeInjectionLine(%q, %q) = %q, want %q", tt.shell, tt.homePath, got, tt.want)
		}
	}
}

func TestOutputCacheDirInjection(t *testing.T) {
	tests := []struct {
		shell      string
		outputPath string
		want       string
	}{
		{
			shell:      "bash",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `export GRAPES_OUT_CACHE_DIR="C:/Users/me/AppData/Roaming/grapes/cache"`,
		},
		{
			shell:      "zsh",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `export GRAPES_OUT_CACHE_DIR="C:/Users/me/AppData/Roaming/grapes/cache"`,
		},
		{
			shell:      "nushell",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `$env.GRAPES_OUT_CACHE_DIR = ($env.GRAPES_OUTPUT_PATH | path join "cache")`,
		},
		{
			shell:      "pwsh",
			outputPath: `C:\Users\me\AppData\Roaming\grapes`,
			want:       `$env:GRAPES_OUT_CACHE_DIR = Join-Path $env:GRAPES_OUTPUT_PATH "cache"`,
		},
	}

	for _, tt := range tests {
		if got := OutputCacheDirInjectionLine(tt.shell, tt.outputPath); got != tt.want {
			t.Errorf("OutputCacheDirInjectionLine(%q, %q) = %q, want %q", tt.shell, tt.outputPath, got, tt.want)
		}
	}
}

func TestPathCleanInjectionLine(t *testing.T) {
	tests := []struct {
		shell    string
		execPath string
		want     string
	}{
		{
			shell:    "bash",
			execPath: `C:\tools\grapes.exe`,
			want:     `if __grapes_path_cleaned="$("C:/tools/grapes.exe" --path-clean "$PATH")"; then export PATH="$__grapes_path_cleaned"; fi; unset __grapes_path_cleaned`,
		},
		{
			shell:    "zsh",
			execPath: `/opt/grapes`,
			want:     `if __grapes_path_cleaned="$("/opt/grapes" --path-clean "$PATH")"; then export PATH="$__grapes_path_cleaned"; fi; unset __grapes_path_cleaned`,
		},
		{
			shell:    "nushell",
			execPath: `/opt/grapes`,
			want:     `let __grapes_path_cleaned = (^'/opt/grapes' --path-clean ($env.PATH | str join (char esep)) | complete); if $__grapes_path_cleaned.exit_code == 0 { $env.PATH = ($__grapes_path_cleaned.stdout | split row (char nl) | get 0 | split row (char esep)) }`,
		},
		{
			shell:    "pwsh",
			execPath: `C:\tools\grapes.exe`,
			want:     `$__grapes_path_cleaned = & 'C:\tools\grapes.exe' --path-clean $env:PATH; if ($? -and $LASTEXITCODE -eq 0) { $env:PATH = $__grapes_path_cleaned }; Remove-Variable __grapes_path_cleaned -ErrorAction SilentlyContinue`,
		},
	}

	for _, tt := range tests {
		got, err := PathCleanInjectionLine(tt.shell, tt.execPath)
		if err != nil {
			t.Fatalf("PathCleanInjectionLine(%q) returned error: %v", tt.shell, err)
		}
		if got != tt.want {
			t.Fatalf("PathCleanInjectionLine(%q) = %q, want %q", tt.shell, got, tt.want)
		}
	}
}

func TestIfdefMatch(t *testing.T) {
	input := "--#ifdef BASH\necho bash\n--#endif\necho common\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("bash") + "echo bash\necho common\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIfdefNoMatch(t *testing.T) {
	input := "--#ifdef BASH\necho bash\n--#endif\necho common\n"
	result, err := Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("zsh") + "echo common\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIfndef(t *testing.T) {
	input := "--#ifndef BASH\necho not-bash\n--#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("bash")
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected = shellLine("zsh") + "echo not-bash\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestElse(t *testing.T) {
	input := "--#ifdef BASH\necho bash\n--#else\necho other\n--#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("bash") + "echo bash\n"
	if result != expected {
		t.Errorf("bash: got %q, want %q", result, expected)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected = shellLine("zsh") + "echo other\n"
	if result != expected {
		t.Errorf("zsh: got %q, want %q", result, expected)
	}
}

func TestElif(t *testing.T) {
	input := "--#ifdef BASH\necho bash\n--#elif ZSH\necho zsh\n--#else\necho other\n--#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != shellLine("bash")+"echo bash\n" {
		t.Errorf("bash: got %q", result)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if result != shellLine("zsh")+"echo zsh\n" {
		t.Errorf("zsh: got %q", result)
	}
}

func TestIfdefPwshAndNushell(t *testing.T) {
	input := "--#ifdef NUSHELL\necho nu\n--#elif PWSH\necho pwsh\n--#else\necho other\n--#endif\n"

	result, err := Process(input, "pwsh")
	if err != nil {
		t.Fatal(err)
	}

	expected := shellLine("pwsh") + "echo pwsh\n"
	if result != expected {
		t.Fatalf("got %q, want %q", result, expected)
	}
}

func TestIfdefNushellAndPwsh(t *testing.T) {
	input := "--#ifdef NUSHELL\necho nu\n--#elif PWSH\necho pwsh\n--#else\necho other\n--#endif\n"

	result, err := Process(input, "nushell")
	if err != nil {
		t.Fatal(err)
	}

	expected := shellLine("nushell") + "echo nu\n"
	if result != expected {
		t.Fatalf("got %q, want %q", result, expected)
	}
}

func TestNestedDirectives(t *testing.T) {
	input := "--#ifdef BASH\n--#ifdef ZSH\necho both\n--#else\necho bash-only\n--#endif\n--#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("bash") + "echo bash-only\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestUnterminatedDirective(t *testing.T) {
	input := "--#ifdef BASH\necho bash\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for unterminated directive")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("error should mention unterminated, got: %s", err.Error())
	}
}

func TestUnknownDirective(t *testing.T) {
	input := "--#ifdef BASH\necho bash\n--#endif\n--#undef FOO\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for unknown directive")
	}
}

func TestMultipleDirectives(t *testing.T) {
	input := "export PATH=/bin\n--#ifdef BASH\nexport BASH_VAR=1\n--#endif\n--#ifdef ZSH\nexport ZSH_VAR=1\n--#endif\necho done\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := shellLine("bash") + "export PATH=/bin\nexport BASH_VAR=1\necho done\n"
	if result != expected {
		t.Errorf("bash: got %q, want %q", result, expected)
	}
}

func TestIfdefWrongArgCount(t *testing.T) {
	input := "--#ifdef A B\necho hi\n--#endif\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for --#ifdef with wrong arg count")
	}
}

func TestIfndefWrongArgCount(t *testing.T) {
	input := "--#ifndef A B\necho hi\n--#endif\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for --#ifndef with wrong arg count")
	}
}

func TestOrphanElif(t *testing.T) {
	input := "--#elif BASH\necho hi\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for orphan --#elif")
	}
}

func TestOrphanElse(t *testing.T) {
	input := "--#else\necho hi\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for orphan --#else")
	}
}

func TestOrphanEndif(t *testing.T) {
	input := "--#endif\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for orphan --#endif")
	}
}

func TestShellCommentsArePreserved(t *testing.T) {
	input := "#!/usr/bin/env bash\n# regular comment\necho hello\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}
