package renderer

import (
	"strings"
	"testing"
)

func TestRenderBlockPowershell(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want string
	}{
		{
			name: "windows path separator",
			goos: "windows",
			want: "$env:FOO = 'bar'\n$env:PATH = '/tool/bin' + ';' + $env:PATH\necho done\n",
		},
		{
			name: "non windows path separator",
			goos: "linux",
			want: "$env:FOO = 'bar'\n$env:PATH = '/tool/bin' + ':' + $env:PATH\necho done\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock(tt.goos, "powershell", map[string]string{
				"FOO": "bar",
			}, []string{"/tool/bin"}, "echo done\n")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockNushell(t *testing.T) {
	got, err := RenderBlock("linux", "nushell", map[string]string{
		"FOO": "bar",
	}, []string{"/tool/bin"}, "echo done\n")
	if err != nil {
		t.Fatal(err)
	}

	want := "$env.FOO = 'bar'\n$env.PATH = ($env.PATH | prepend '/tool/bin')\necho done\n"
	if got != want {
		t.Fatalf("RenderBlock() = %q, want %q", got, want)
	}
}

func TestRenderBlockPreservesSortedEnvOrder(t *testing.T) {
	got, err := RenderBlock("linux", "bash", map[string]string{
		"ZED":   "2",
		"ALPHA": "1",
	}, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	want := "export ALPHA=\"1\"\nexport ZED=\"2\"\n"
	if got != want {
		t.Fatalf("RenderBlock() = %q, want %q", got, want)
	}
}

func TestRenderBlockPreservesBashAndZshExpansionOrientedValues(t *testing.T) {
	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock("linux", tt.shell, map[string]string{
				"GOPATH": `${GOPATH:-$HOME/go}`,
			}, []string{`$GOPATH/bin`}, "")
			if err != nil {
				t.Fatal(err)
			}

			want := "export GOPATH=\"${GOPATH:-$HOME/go}\"\nexport PATH=\"$GOPATH/bin\":$PATH\n"
			if got != want {
				t.Fatalf("RenderBlock() = %q, want %q", got, want)
			}
		})
	}
}

func TestRenderBlockPreservesBashAndZshBackslashDollarSemantics(t *testing.T) {
	tests := []struct {
		name   string
		shell  string
		env    map[string]string
		paths  []string
		want   string
	}{
		{
			name:  "bash",
			shell: "bash",
			env: map[string]string{
				"LITERAL_HOME":   `\$HOME`,
				"LITERAL_GOPATH": `\${GOPATH}`,
			},
			paths: []string{`$HOME/\$APP/bin`},
			want: "export LITERAL_GOPATH=\"\\${GOPATH}\"\nexport LITERAL_HOME=\"\\$HOME\"\nexport PATH=\"$HOME/\\$APP/bin\":$PATH\n",
		},
		{
			name:  "zsh",
			shell: "zsh",
			env: map[string]string{
				"LITERAL_HOME":   `\$HOME`,
				"LITERAL_GOPATH": `\${GOPATH}`,
			},
			paths: []string{`$HOME/\$APP/bin`},
			want: "export LITERAL_GOPATH=\"\\${GOPATH}\"\nexport LITERAL_HOME=\"\\$HOME\"\nexport PATH=\"$HOME/\\$APP/bin\":$PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock("linux", tt.shell, tt.env, tt.paths, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockEscapesTrailingBackslashesForBashAndZsh(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  string
	}{
		{
			name:  "bash",
			shell: "bash",
			want:  "export FOO=\"foo\\\\\"\nexport PATH=\"/tool\\\\\":$PATH\n",
		},
		{
			name:  "zsh",
			shell: "zsh",
			want:  "export FOO=\"foo\\\\\"\nexport PATH=\"/tool\\\\\":$PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock("linux", tt.shell, map[string]string{
				"FOO": `foo\`,
			}, []string{`/tool\`}, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockEscapesBackticksForBashAndZsh(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  string
	}{
		{
			name:  "bash",
			shell: "bash",
			want:  "export FOO=\"foo\\`bar\\`\"\nexport PATH=\"/tool/\\`bin\\`\":$PATH\n",
		},
		{
			name:  "zsh",
			shell: "zsh",
			want:  "export FOO=\"foo\\`bar\\`\"\nexport PATH=\"/tool/\\`bin\\`\":$PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock("linux", tt.shell, map[string]string{
				"FOO": "foo`bar`",
			}, []string{"/tool/`bin`"}, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockEscapesBackslashesBeforeBackticksForBashAndZsh(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  string
	}{
		{
			name:  "bash",
			shell: "bash",
			want:  "export FOO=\"foo\\\\\\`bar\\\\\\\\\\`baz\"\nexport PATH=\"/tool\\\\\\`bin\\\\\\\\\\`\":$PATH\n",
		},
		{
			name:  "zsh",
			shell: "zsh",
			want:  "export FOO=\"foo\\\\\\`bar\\\\\\\\\\`baz\"\nexport PATH=\"/tool\\\\\\`bin\\\\\\\\\\`\":$PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock("linux", tt.shell, map[string]string{
				"FOO": "foo\\`bar\\\\`baz",
			}, []string{"/tool\\`bin\\\\`"}, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockEscapesBackslashesBeforeNewlinesForBashAndZsh(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  string
	}{
		{
			name:  "bash",
			shell: "bash",
			want:  "export FOO=\"foo\\\\\nbar\"\nexport PATH=\"/tool\\\\\nbar\":$PATH\n",
		},
		{
			name:  "zsh",
			shell: "zsh",
			want:  "export FOO=\"foo\\\\\nbar\"\nexport PATH=\"/tool\\\\\nbar\":$PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock("linux", tt.shell, map[string]string{
				"FOO": "foo\\\nbar",
			}, []string{"/tool\\\nbar"}, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockEscapesEmbeddedDoubleQuotes(t *testing.T) {
	tests := []struct {
		name  string
		goos  string
		shell string
		want  string
	}{
		{
			name:  "bash",
			goos:  "linux",
			shell: "bash",
			want:  "export FOO=\"a\\\"b\"\nexport PATH=\"/tool/\\\"bin\":$PATH\n",
		},
		{
			name:  "zsh",
			goos:  "linux",
			shell: "zsh",
			want:  "export FOO=\"a\\\"b\"\nexport PATH=\"/tool/\\\"bin\":$PATH\n",
		},
		{
			name:  "nushell",
			goos:  "linux",
			shell: "nushell",
			want:  "$env.FOO = 'a\"b'\n$env.PATH = ($env.PATH | prepend '/tool/\"bin')\n",
		},
		{
			name:  "powershell windows",
			goos:  "windows",
			shell: "powershell",
			want:  "$env:FOO = 'a\"b'\n$env:PATH = '/tool/\"bin' + ';' + $env:PATH\n",
		},
		{
			name:  "powershell unix",
			goos:  "linux",
			shell: "powershell",
			want:  "$env:FOO = 'a\"b'\n$env:PATH = '/tool/\"bin' + ':' + $env:PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock(tt.goos, tt.shell, map[string]string{
				"FOO": `a"b`,
			}, []string{`/tool/"bin`}, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockEscapesEmbeddedSingleQuotes(t *testing.T) {
	tests := []struct {
		name  string
		goos  string
		shell string
		want  string
	}{
		{
			name:  "bash",
			goos:  "linux",
			shell: "bash",
			want:  "export FOO=\"a'b\"\nexport PATH=\"/tool/'bin\":$PATH\n",
		},
		{
			name:  "zsh",
			goos:  "linux",
			shell: "zsh",
			want:  "export FOO=\"a'b\"\nexport PATH=\"/tool/'bin\":$PATH\n",
		},
		{
			name:  "nushell",
			goos:  "linux",
			shell: "nushell",
			want:  "$env.FOO = 'a''b'\n$env.PATH = ($env.PATH | prepend '/tool/''bin')\n",
		},
		{
			name:  "powershell windows",
			goos:  "windows",
			shell: "powershell",
			want:  "$env:FOO = 'a''b'\n$env:PATH = '/tool/''bin' + ';' + $env:PATH\n",
		},
		{
			name:  "powershell unix",
			goos:  "linux",
			shell: "powershell",
			want:  "$env:FOO = 'a''b'\n$env:PATH = '/tool/''bin' + ':' + $env:PATH\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderBlock(tt.goos, tt.shell, map[string]string{
				"FOO": "a'b",
			}, []string{"/tool/'bin"}, "")
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("RenderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockPreservesPathOrderAtRuntime(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		shell       string
		initialPath string
		wantPath    string
	}{
		{name: "bash", goos: "linux", shell: "bash", initialPath: "base", wantPath: "/one:/two:base"},
		{name: "zsh", goos: "linux", shell: "zsh", initialPath: "base", wantPath: "/one:/two:base"},
		{name: "nushell", goos: "linux", shell: "nushell", initialPath: "base", wantPath: "/one:/two:base"},
		{name: "powershell windows", goos: "windows", shell: "powershell", initialPath: "base", wantPath: "/one;/two;base"},
		{name: "powershell unix", goos: "linux", shell: "powershell", initialPath: "base", wantPath: "/one:/two:base"},
		{name: "bash embedded single quote", goos: "linux", shell: "bash", initialPath: "base", wantPath: "/o'ne:/tw'o:base"},
		{name: "zsh embedded single quote", goos: "linux", shell: "zsh", initialPath: "base", wantPath: "/o'ne:/tw'o:base"},
		{name: "nushell embedded single quote", goos: "linux", shell: "nushell", initialPath: "base", wantPath: "/o'ne:/tw'o:base"},
		{name: "powershell windows embedded single quote", goos: "windows", shell: "powershell", initialPath: "base", wantPath: "/o'ne;/tw'o;base"},
		{name: "powershell unix embedded single quote", goos: "linux", shell: "powershell", initialPath: "base", wantPath: "/o'ne:/tw'o:base"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := []string{"/one", "/two"}
			if strings.Contains(tt.name, "embedded single quote") {
				paths = []string{"/o'ne", "/tw'o"}
			}

			got, err := RenderBlock(tt.goos, tt.shell, nil, paths, "")
			if err != nil {
				t.Fatal(err)
			}

			finalPath := evaluateRenderedPath(t, tt.goos, tt.shell, got, tt.initialPath)
			if finalPath != tt.wantPath {
				t.Fatalf("final PATH = %q, want %q\nrendered:\n%s", finalPath, tt.wantPath, got)
			}
		})
	}
}

func TestRenderBlockRejectsUnsupportedShellWithoutEnvPathsOrBody(t *testing.T) {
	_, err := RenderBlock("linux", "fish", nil, nil, "")
	if err == nil {
		t.Fatal("RenderBlock() error = nil, want unsupported shell error")
	}

	want := `unsupported shell "fish"`
	if err.Error() != want {
		t.Fatalf("RenderBlock() error = %q, want %q", err.Error(), want)
	}
}

func TestRenderBlockRejectsUnsupportedShellWithoutEnvPathsWithBody(t *testing.T) {
	_, err := RenderBlock("linux", "fish", nil, nil, "echo hi\n")
	if err == nil {
		t.Fatal("RenderBlock() error = nil, want unsupported shell error")
	}

	want := `unsupported shell "fish"`
	if err.Error() != want {
		t.Fatalf("RenderBlock() error = %q, want %q", err.Error(), want)
	}
}

func TestRenderBlockPreservesBodyPassthrough(t *testing.T) {
	body := "echo one\n\n\n"

	got, err := RenderBlock("linux", "bash", map[string]string{
		"FOO": "bar",
	}, nil, body)
	if err != nil {
		t.Fatal(err)
	}

	want := "export FOO=\"bar\"\n" + body
	if got != want {
		t.Fatalf("RenderBlock() = %q, want %q", got, want)
	}
}

func evaluateRenderedPath(t *testing.T, goos, shell, rendered, initialPath string) string {
	t.Helper()

	path := initialPath
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	for _, line := range lines {
		switch shell {
		case "bash", "zsh":
			const prefix = `export PATH=`
			if strings.HasPrefix(line, prefix) {
				path = evaluatePosixPathPrepend(t, strings.TrimPrefix(line, prefix), path)
			}
		case "nushell":
			const prefix = `$env.PATH = ($env.PATH | prepend `
			if strings.HasPrefix(line, prefix) {
				path = evaluateNushellPathPrepend(t, strings.TrimPrefix(line, prefix), path)
			}
		case "powershell":
			const prefix = `$env:PATH = `
			if strings.HasPrefix(line, prefix) {
				path = evaluatePowershellPrepend(t, goos, strings.TrimPrefix(line, prefix), path)
			}
		default:
			t.Fatalf("unsupported shell %q", shell)
		}
	}

	return path
}

func evaluatePosixPathPrepend(t *testing.T, expr, currentPath string) string {
	t.Helper()

	if strings.HasSuffix(expr, `:$PATH"`) {
		return strings.TrimSuffix(strings.TrimPrefix(expr, `"`), `:$PATH"`) + ":" + currentPath
	}

	literal, rest := parseQuotedLiteral(t, expr)
	if rest != ":$PATH" {
		t.Fatalf("PATH expression %q missing suffix %q", expr, ":$PATH")
	}

	return literal + ":" + currentPath
}

func evaluateNushellPathPrepend(t *testing.T, expr, currentPath string) string {
	t.Helper()

	if strings.HasSuffix(expr, `")`) {
		return strings.TrimSuffix(strings.TrimPrefix(expr, `"`), `")`) + ":" + currentPath
	}

	literal, rest := parseQuotedLiteral(t, expr)
	if rest != ")" {
		t.Fatalf("PATH expression %q missing suffix %q", expr, ")")
	}

	return literal + ":" + currentPath
}

func evaluatePowershellPrepend(t *testing.T, goos, expr, currentPath string) string {
	t.Helper()

	separator := ":"
	suffix := ` + ':' + $env:PATH`
	if goos == "windows" {
		separator = ";"
		suffix = ` + ';' + $env:PATH`
	}

	if strings.HasSuffix(expr, `:$env:PATH"`) {
		return strings.TrimSuffix(strings.TrimPrefix(expr, `"`), `:$env:PATH"`) + separator + currentPath
	}

	literal, rest := parseQuotedLiteral(t, expr)
	if rest != suffix {
		t.Fatalf("PATH expression %q missing suffix %q", expr, suffix)
	}

	return literal + separator + currentPath
}

func parseQuotedLiteral(t *testing.T, expr string) (string, string) {
	t.Helper()

	if expr == "" {
		t.Fatal("empty expression")
	}

	var value strings.Builder
	i := 0
	for i < len(expr) {
		switch expr[i] {
		case '\'':
			fragment, next := parseSingleQuotedFragment(t, expr, i)
			value.WriteString(fragment)
			i = next
		case '"':
			fragment, next := parseDoubleQuotedFragment(t, expr, i)
			value.WriteString(fragment)
			i = next
		default:
			if i == 0 {
				t.Fatalf("unsupported quoted literal in %q", expr)
			}
			return value.String(), expr[i:]
		}
	}

	return value.String(), ""
}

func parseSingleQuotedFragment(t *testing.T, expr string, start int) (string, int) {
	t.Helper()

	var value strings.Builder
	for i := start + 1; i < len(expr); i++ {
		if expr[i] != '\'' {
			value.WriteByte(expr[i])
			continue
		}

		if i+1 < len(expr) && expr[i+1] == '\'' {
			value.WriteByte('\'')
			i++
			continue
		}

		return value.String(), i + 1
	}

	t.Fatalf("unterminated single-quoted literal in %q", expr)
	return "", 0
}

func parseDoubleQuotedFragment(t *testing.T, expr string, start int) (string, int) {
	t.Helper()

	for i := start + 1; i < len(expr); i++ {
		if expr[i] == '"' {
			return expr[start+1 : i], i + 1
		}
	}

	t.Fatalf("unterminated double-quoted literal in %q", expr)
	return "", 0
}
