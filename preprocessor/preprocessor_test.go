package preprocessor

import (
	"strings"
	"testing"
)

func TestNoDirectives(t *testing.T) {
	input := "export FOO=bar\necho hello\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}

func TestIfdefMatch(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#endif\necho common\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo bash\necho common\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIfdefNoMatch(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#endif\necho common\n"
	result, err := Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo common\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIfndef(t *testing.T) {
	input := "#ifndef BASH\necho not-bash\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo not-bash\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestElse(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#else\necho other\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo bash\n"
	if result != expected {
		t.Errorf("bash: got %q, want %q", result, expected)
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	expected = "echo other\n"
	if result != expected {
		t.Errorf("zsh: got %q, want %q", result, expected)
	}
}

func TestElif(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#elif ZSH\necho zsh\n#else\necho other\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo bash\n" {
		t.Errorf("bash: got %q, want %q", result, "echo bash\n")
	}

	result, err = Process(input, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo zsh\n" {
		t.Errorf("zsh: got %q, want %q", result, "echo zsh\n")
	}
}

func TestNestedDirectives(t *testing.T) {
	input := "#ifdef BASH\n#ifdef ZSH\necho both\n#else\necho bash-only\n#endif\n#endif\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "echo bash-only\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestUnterminatedDirective(t *testing.T) {
	input := "#ifdef BASH\necho bash\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for unterminated directive")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("error should mention unterminated, got: %s", err.Error())
	}
}

func TestUnknownDirective(t *testing.T) {
	input := "#ifdef BASH\necho bash\n#endif\n#undef FOO\n"
	_, err := Process(input, "bash")
	if err == nil {
		t.Error("expected error for unknown directive")
	}
}

func TestMultipleDirectives(t *testing.T) {
	input := "export PATH=/bin\n#ifdef BASH\nexport BASH_VAR=1\n#endif\n#ifdef ZSH\nexport ZSH_VAR=1\n#endif\necho done\n"
	result, err := Process(input, "bash")
	if err != nil {
		t.Fatal(err)
	}
	expected := "export PATH=/bin\nexport BASH_VAR=1\necho done\n"
	if result != expected {
		t.Errorf("bash: got %q, want %q", result, expected)
	}
}
