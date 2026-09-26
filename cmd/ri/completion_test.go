package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCompletionUnsupportedShell(t *testing.T) {
	for _, shell := range []string{"fish", "", "BASH"} {
		code, stdout, stderr := runCLI(t, "-completion", shell)
		if code != 2 {
			t.Errorf("-completion %q: exit code = %d, want 2", shell, code)
		}
		if stdout != "" || !strings.Contains(stderr, "unsupported shell") {
			t.Errorf("-completion %q: stdout = %q, stderr = %q", shell, stdout, stderr)
		}
	}
}

// helpFlagNames returns every flag the CLI defines, as listed by -h.
func helpFlagNames(t *testing.T) []string {
	t.Helper()
	_, _, help := runCLI(t, "-h")
	matches := regexp.MustCompile(`(?m)^  -([a-z-]+)`).FindAllStringSubmatch(help, -1)
	if len(matches) < 5 {
		t.Fatalf("could not find the flags in the help output:\n%s", help)
	}

	var names []string
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

// The scripts are generated from the flag set, so a new flag or repo type must show up in them.
func TestCompletionCoversFlagsAndTypes(t *testing.T) {
	flags := helpFlagNames(t)

	for _, shell := range completionShells {
		t.Run(shell, func(t *testing.T) {
			code, script, stderr := runCLI(t, "-completion", shell)
			if code != 0 {
				t.Fatalf("exit code = %d, stderr: %s", code, stderr)
			}
			for _, f := range flags {
				if !strings.Contains(script, "-"+f) {
					t.Errorf("script does not mention flag -%s", f)
				}
			}
			for _, typ := range typeList() {
				if !strings.Contains(script, typ) {
					t.Errorf("script does not mention repo type %s", typ)
				}
			}
		})
	}
}

// The scripts must be valid syntax for the shell they target.
func TestCompletionScriptSyntax(t *testing.T) {
	for _, shell := range completionShells {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			_, script, _ := runCLI(t, "-completion", shell)

			cmd := exec.Command(path, "-n")
			cmd.Stdin = strings.NewReader(script)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s -n rejected the script: %v\n%s", shell, err, out)
			}
		})
	}
}

func TestZshEscape(t *testing.T) {
	got := zshEscape(`don't [break]: here`)
	if want := `don'\''t \[break\]\: here`; got != want {
		t.Errorf("zshEscape = %q, want %q", got, want)
	}
}

// Runs the bash script for real: sets up a completion context and calls the completion function.
func TestBashCompletionResults(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}

	dir := t.TempDir()
	for _, d := range []string{"projects", "dir with space"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"a.yaml", "b.yml", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)

	_, script, _ := runCLI(t, "-completion", "bash")

	// complete runs _ri with the given words (the last one is being completed), and prints one candidate per line.
	complete := func(words ...string) []string {
		quoted := make([]string, len(words))
		for i, w := range words {
			quoted[i] = "'" + w + "'"
		}
		driver := script + "\nCOMP_WORDS=(" + strings.Join(quoted, " ") + ")\n" +
			"COMP_CWORD=" + string(rune('0'+len(words)-1)) + "\n_ri\nprintf '%s\\n' \"${COMPREPLY[@]}\"\n"
		out, err := exec.Command(bash, "-c", driver).CombinedOutput()
		if err != nil {
			t.Fatalf("bash failed: %v\n%s", err, out)
		}
		return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' })
	}

	tests := []struct {
		words []string
		want  []string // exact set of candidates
	}{
		{[]string{"ri", "-type", "k"}, []string{"kube_app"}},
		{[]string{"ri", "-type", "p"}, []string{"python", "packer"}},
		{[]string{"ri", "-completion", ""}, []string{"bash", "zsh"}},
		{[]string{"ri", "-s"}, []string{"-skip-commands"}},
		{[]string{"ri", "-f"}, []string{"-force"}},
		{[]string{"ri", "-name", "d"}, []string{"dir with space"}},
		{[]string{"ri", "-name", "pro"}, []string{"projects"}},
		{[]string{"ri", "-yaml", "a"}, []string{"a.yaml"}},
		{[]string{"ri", "-yaml", "n"}, nil}, // notes.txt is not a YAML file
	}
	for _, tc := range tests {
		got := complete(tc.words...)
		if !sameSet(got, tc.want) {
			t.Errorf("complete %q = %q, want %q", tc.words, got, tc.want)
		}
	}

	// a -yaml word offers directories and YAML files, but no other files
	got := strings.Join(complete("ri", "-yaml", ""), "|")
	for _, want := range []string{"a.yaml", "b.yml", "projects", "dir with space"} {
		if !strings.Contains(got, want) {
			t.Errorf("complete -yaml \"\" = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "notes.txt") {
		t.Errorf("complete -yaml \"\" offered a non-YAML file: %q", got)
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}
