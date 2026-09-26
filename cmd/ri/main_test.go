package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI runs the CLI in a fresh working directory and returns its exit code and output.
func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestListTypes(t *testing.T) {
	code, stdout, _ := runCLI(t, "-list-types")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"go", "c", "python", "ansible", "kube_app", "terraform", "packer", "custom"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("-list-types output %q is missing %q", stdout, want)
		}
	}
}

func TestHelpExitsZero(t *testing.T) {
	code, _, stderr := runCLI(t, "-h")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr, "-force") {
		t.Errorf("help does not mention -force:\n%s", stderr)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := map[string]struct {
		args []string
		want string // substring of the error message
	}{
		"no arguments":      {nil, "-type is required"},
		"missing type":      {[]string{"-name", "x"}, "-type is required"},
		"missing name":      {[]string{"-type", "go"}, "-name is required"},
		"unknown type":      {[]string{"-type", "nope", "-name", "x"}, "invalid repo flag"},
		"unknown flag":      {[]string{"-bogus"}, "flag provided but not defined"},
		"stray argument":    {[]string{"-type", "go", "-name", "x", "extra"}, "unexpected arguments: extra"},
		"custom needs yaml": {[]string{"-type", "custom", "-name", "x"}, "YAML file is required"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			code, _, stderr := runCLI(t, tc.args...)
			if code == 0 {
				t.Errorf("exit code = 0, want non-zero")
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tc.want)
			}
			entries, _ := os.ReadDir(".")
			if len(entries) != 0 {
				t.Errorf("files were created despite the error: %v", entries)
			}
		})
	}
}

func TestCreatesRepo(t *testing.T) {
	t.Chdir(t.TempDir())
	code, _, stderr := runCLI(t, "-type", "go", "-name", "nested/proj", "-skip-commands")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join("nested", "proj", "go.mod")); err != nil {
		t.Error(err)
	}
	// -skip-commands means no `git init`
	if _, err := os.Stat(filepath.Join("nested", "proj", ".git")); err == nil {
		t.Error("commands ran despite -skip-commands")
	}
}

func TestRefusesToOverwriteUnlessForced(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("README.md", []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runCLI(t, "-type", "c", "-name", ".", "-skip-commands")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "README.md") || !strings.Contains(stderr, "-force") {
		t.Errorf("stderr should name the file and mention -force: %q", stderr)
	}
	if got, _ := os.ReadFile("README.md"); string(got) != "keep me" {
		t.Errorf("README.md was overwritten: %q", got)
	}

	code, _, stderr = runCLI(t, "-type", "c", "-name", ".", "-skip-commands", "-force")
	if code != 0 {
		t.Fatalf("with -force: exit code = %d, stderr: %s", code, stderr)
	}
	if got, _ := os.ReadFile("README.md"); string(got) == "keep me" {
		t.Error("-force did not overwrite README.md")
	}
}

func TestCustomTemplateFile(t *testing.T) {
	t.Chdir(t.TempDir())
	tmpl := filepath.Join(t.TempDir(), "t.yaml")
	if err := os.WriteFile(tmpl, []byte("dirs:\n  - path: \"./\"\n    files:\n      - name: hello.txt\n        content: 'hello {{ .Name }}'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runCLI(t, "-type", "custom", "-yaml", tmpl, "-name", "out/proj", "-skip-commands")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if got, _ := os.ReadFile(filepath.Join("out", "proj", "hello.txt")); string(got) != "hello proj" {
		t.Errorf("hello.txt = %q", got)
	}

	code, _, stderr = runCLI(t, "-type", "custom", "-yaml", filepath.Join(t.TempDir(), "nope.yaml"), "-name", "x")
	if code != 1 || !strings.Contains(stderr, "ri: loading") {
		t.Errorf("missing yaml: code = %d, stderr = %q", code, stderr)
	}
}

func TestRequiredCommandFailureIsAnError(t *testing.T) {
	t.Chdir(t.TempDir())
	tmpl := filepath.Join(t.TempDir(), "t.yaml")
	if err := os.WriteFile(tmpl, []byte("commands:\n  - name: ri-no-such-command\n"), 0644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runCLI(t, "-type", "custom", "-yaml", tmpl, "-name", "proj")
	if code != 1 || !strings.Contains(stderr, "not found in PATH") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}
