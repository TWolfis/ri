package ri_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TWolfis/ri"
)

const customYAML = `---
name: custom_repo

dirs:
  - path: "src"
  - path: "docs"
    files:
      - name: "README.md"
        content: |
          # {{ .Name }}
`

// repoCase describes how to build and verify a repo for one RepoFlag.
type repoCase struct {
	// yamlFile returns the yaml file to pass to NewRepo, or nil if the flag does not need one.
	yamlFile func(t *testing.T) *string
	// wantPaths are the paths, relative to the repo root, that Init must create.
	wantPaths []string
	// wantContents maps a path relative to the repo root to the rendered content it must have.
	// Each value is a func of the repo root, whose base is what {{ .Name }} renders to.
	wantContents map[string]func(root string) string
}

// repoCases must contain an entry for every flag yielded by RepoFlags, so adding a
// new flag without extending this table fails TestNewRepoInit.
var repoCases = map[ri.RepoFlag]repoCase{
	ri.GoRepoFlag: {
		wantPaths: []string{
			filepath.Join("cmd", "app", "main.go"), "internal", "pkg",
			"go.mod", "Makefile", "README.md", "CLAUDE.md", ".gitignore",
			filepath.Join(".github", "workflows", "ci.yml"),
			filepath.Join(".github", "workflows", "dependabot-automerge.yml"),
			filepath.Join(".github", "dependabot.yml"),
		},
		wantContents: map[string]func(string) string{
			"go.mod": func(root string) string { return "module " + filepath.Base(root) + "\n\ngo 1.27\n" },
		},
	},
	ri.CRepoFlag: {
		wantPaths: []string{
			filepath.Join("src", "main.c"), "include",
			"Makefile", ".clang-format", "README.md", "CLAUDE.md", ".gitignore",
		},
	},
	ri.PythonRepoFlag: {
		wantPaths: []string{
			filepath.Join("src", "app", "__init__.py"), filepath.Join("src", "app", "main.py"),
			filepath.Join("tests", "test_main.py"),
			"pyproject.toml", "README.md", "CLAUDE.md", ".gitignore",
		},
	},
	ri.AnsibleRepoFlag: {
		wantPaths: []string{
			filepath.Join("inventory", "hosts.yml"),
			filepath.Join("inventory", "group_vars", "all.yml"),
			filepath.Join("playbooks", "site.yml"),
			filepath.Join("roles", "common", "tasks", "main.yml"),
			"ansible.cfg", "requirements.yml", "README.md", "CLAUDE.md", ".gitignore",
		},
	},
	ri.KubeAppRepoFlag: {
		wantPaths: []string{
			filepath.Join("base", "kustomization.yaml"), filepath.Join("base", "deployment.yaml"),
			filepath.Join("base", "service.yaml"),
			filepath.Join("overlays", "dev", "kustomization.yaml"),
			filepath.Join("overlays", "prod", "kustomization.yaml"),
			filepath.Join("overlays", "prod", "pdb.yaml"),
			filepath.Join("argocd", "project.yaml"),
			filepath.Join("argocd", "applications", "dev.yaml"),
			filepath.Join("argocd", "applications", "prod.yaml"),
			"Makefile", "README.md", "CLAUDE.md", ".gitignore",
		},
		wantContents: map[string]func(string) string{
			filepath.Join("overlays", "dev", "kustomization.yaml"): func(root string) string {
				name := filepath.Base(root)
				return "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nnamespace: " + name + "-dev\n" +
					"resources:\n  - ../../base\nlabels:\n  - pairs:\n      app.kubernetes.io/environment: dev\n" +
					"replicas:\n  - name: " + name + "\n    count: 1\n"
			},
		},
	},
	ri.TerraformRepoFlag: {
		wantPaths: []string{
			filepath.Join("modules", "example", "main.tf"),
			filepath.Join("modules", "example", "variables.tf"),
			filepath.Join("modules", "example", "outputs.tf"),
			filepath.Join("envs", "dev.tfvars"), filepath.Join("envs", "prod.tfvars"),
			"main.tf", "variables.tf", "outputs.tf", "versions.tf",
			"Makefile", "README.md", "CLAUDE.md", ".gitignore",
		},
	},
	ri.PackerRepoFlag: {
		wantPaths: []string{
			filepath.Join("scripts", "provision.sh"),
			"packer.pkr.hcl", "variables.pkr.hcl", "sources.pkr.hcl", "build.pkr.hcl",
			"example.pkrvars.hcl", "Makefile", "README.md", "CLAUDE.md", ".gitignore",
		},
	},
	ri.CustomRepoFlag: {
		yamlFile: func(t *testing.T) *string {
			path := filepath.Join(t.TempDir(), "custom.yaml")
			if err := os.WriteFile(path, []byte(customYAML), 0644); err != nil {
				t.Fatal(err)
			}
			return &path
		},
		wantPaths: []string{"src", "docs", filepath.Join("docs", "README.md")},
		wantContents: map[string]func(string) string{
			filepath.Join("docs", "README.md"): func(root string) string { return "# " + filepath.Base(root) + "\n" },
		},
	},
}

func TestRepoFlags(t *testing.T) {
	flags := slices.Collect(ri.RepoFlags())

	if len(flags) != len(repoCases) {
		t.Errorf("RepoFlags yielded %d flags, repoCases has %d entries", len(flags), len(repoCases))
	}

	seen := map[ri.RepoFlag]bool{}
	for _, f := range flags {
		if seen[f] {
			t.Errorf("RepoFlags yielded %q twice", f)
		}
		seen[f] = true
	}
}

func TestRepoFlagStringSetRoundTrip(t *testing.T) {
	for f := range ri.RepoFlags() {
		t.Run(f.String(), func(t *testing.T) {
			if f.String() == "unknown" {
				t.Fatalf("flag %d has no String() name", int(f))
			}

			var got ri.RepoFlag
			if err := got.Set(f.String()); err != nil {
				t.Fatalf("Set(%q): %v", f.String(), err)
			}
			if got != f {
				t.Errorf("Set(%q) = %d, want %d", f.String(), got, f)
			}
		})
	}
}

func TestRepoFlagSetInvalid(t *testing.T) {
	var f ri.RepoFlag
	for _, v := range []string{"", "rust", "Go", "unknown"} {
		if err := f.Set(v); err == nil {
			t.Errorf("Set(%q) succeeded, want error", v)
		}
	}
}

func TestRepoFlagsStopsEarly(t *testing.T) {
	count := 0
	for range ri.RepoFlags() {
		count++
		break
	}
	if count != 1 {
		t.Errorf("iterated %d times after break, want 1", count)
	}
}

func TestNewRepoInit(t *testing.T) {
	for f := range ri.RepoFlags() {
		t.Run(f.String(), func(t *testing.T) {
			tc, ok := repoCases[f]
			if !ok {
				t.Fatalf("no test case for flag %q; add it to repoCases", f)
			}

			var yamlFile *string
			if tc.yamlFile != nil {
				yamlFile = tc.yamlFile(t)
			}

			root := filepath.Join(t.TempDir(), "myrepo")
			repo, err := ri.NewRepo(f, root, yamlFile)
			if err != nil {
				t.Fatalf("NewRepo: %v", err)
			}
			if err := repo.Init(); err != nil {
				t.Fatalf("Init: %v", err)
			}

			for _, p := range tc.wantPaths {
				if _, err := os.Stat(filepath.Join(root, p)); err != nil {
					t.Errorf("expected %s to exist under repo root: %v", p, err)
				}
			}

			for p, want := range tc.wantContents {
				got, err := os.ReadFile(filepath.Join(root, p))
				if err != nil {
					t.Errorf("reading %s: %v", p, err)
					continue
				}
				if string(got) != want(root) {
					t.Errorf("%s content = %q, want %q", p, got, want(root))
				}
			}
		})
	}
}

func TestInitInvalidTemplate(t *testing.T) {
	for name, content := range map[string]string{
		"parse error":   "{{ .Name",
		"unknown field": "{{ .ModuleName }}",
	} {
		t.Run(name, func(t *testing.T) {
			data := "dirs:\n  - path: \"./\"\n    files:\n      - name: f\n        content: '" + content + "'\n"
			parsed, err := ri.FromYAML([]byte(data))
			if err != nil {
				t.Fatal(err)
			}
			parsed.Name = filepath.Join(t.TempDir(), "myrepo")
			if err := parsed.Init(); err == nil {
				t.Error("Init succeeded, want template error")
			}
		})
	}
}

func TestNewRepoCustomErrors(t *testing.T) {
	empty := ""
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(bad, []byte("dirs: [unterminated"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := map[string]*string{"nil yaml": nil, "empty yaml path": &empty, "missing file": &missing, "invalid yaml": &bad}
	for name, yamlFile := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ri.NewRepo(ri.CustomRepoFlag, "x", yamlFile); err == nil {
				t.Error("NewRepo succeeded, want error")
			}
		})
	}
}

func TestNewRepoInvalidFlag(t *testing.T) {
	// -1 is below the first flag, 99 is past the last (including the unexported sentinel)
	for _, f := range []ri.RepoFlag{-1, 99} {
		if _, err := ri.NewRepo(f, "x", nil); err == nil {
			t.Errorf("NewRepo(%d) succeeded, want error", int(f))
		}
	}
}

func TestInitRefusesToOverwrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myrepo")
	existing := filepath.Join(root, "go.mod")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	repo, err := ri.NewRepo(ri.GoRepoFlag, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Init()
	if !errors.Is(err, ri.ErrFilesExist) {
		t.Fatalf("Init error = %v, want ErrFilesExist", err)
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("error should name the conflicting file: %v", err)
	}

	got, _ := os.ReadFile(existing)
	if string(got) != "keep me" {
		t.Errorf("existing file was modified: %q", got)
	}
	// nothing else may be written when Init refuses
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Errorf("Init wrote files despite refusing: %v", entries)
	}
}

func TestInitWithOverwrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myrepo")
	existing := filepath.Join(root, "go.mod")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("old content that is much longer than the template output, to catch missing truncation"), 0644); err != nil {
		t.Fatal(err)
	}

	repo, err := ri.NewRepo(ri.GoRepoFlag, root, nil, ri.WithOverwrite())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	got, _ := os.ReadFile(existing)
	if want := "module myrepo\n\ngo 1.27\n"; string(got) != want {
		t.Errorf("go.mod = %q, want %q", got, want)
	}
}

func TestInitTwiceIsRefused(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myrepo")
	for i, wantErr := range []bool{false, true} {
		repo, err := ri.NewRepo(ri.CRepoFlag, root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Init(); (err != nil) != wantErr {
			t.Errorf("run %d: Init error = %v, wantErr %v", i+1, err, wantErr)
		}
	}
}

func TestInitWritesNothingOnTemplateError(t *testing.T) {
	t.Chdir(t.TempDir())
	// the good file comes first, so a write-as-you-go implementation would leave it behind
	err := initFromYAML(t, "proj", "dirs:\n  - path: \"./\"\n    files:\n      - name: good.txt\n        content: ok\n      - name: bad.txt\n        content: '{{ .Nope }}'\n")
	if err == nil {
		t.Fatal("Init succeeded, want template error")
	}
	if _, err := os.Stat("proj"); err == nil {
		t.Error("Init created files before finding the template error")
	}
}

func TestInitRejectsBrokenCommandArgsBeforeWriting(t *testing.T) {
	t.Chdir(t.TempDir())
	err := initFromYAML(t, "proj", "dirs:\n  - path: \"./\"\n    files:\n      - name: a.txt\n        content: ok\ncommands:\n  - name: echo\n    args: ['{{ .Nope }}']\n")
	if err == nil {
		t.Fatal("Init succeeded, want template error")
	}
	if _, err := os.Stat("proj"); err == nil {
		t.Error("Init created files before finding the broken command arg")
	}
}

func TestNamePattern(t *testing.T) {
	const data = "name_pattern: '^[a-z][a-z0-9-]*$'\ndirs:\n  - path: \"./\"\n    files:\n      - name: f\n        content: x\n"

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"my-app", false},
		{"nested/path/my-app", false}, // only the base is checked
		{"My_App", true},
		{"nested/My_App", true},
		{"1app", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			err := initFromYAML(t, tc.name, data)
			if (err != nil) != tc.wantErr {
				t.Errorf("Init(%q) error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
			if tc.wantErr {
				if _, statErr := os.Stat(tc.name); statErr == nil {
					t.Error("files were created for an invalid name")
				}
			}
		})
	}
}

func TestNamePatternInvalidRegexp(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := initFromYAML(t, "proj", "name_pattern: '['\n"); err == nil {
		t.Error("Init succeeded with an invalid name_pattern")
	}
}

func TestKubeAppRejectsInvalidKubernetesNames(t *testing.T) {
	for name, wantErr := range map[string]bool{
		"my-shop":               false,
		"shop2":                 false,
		"My-Shop":               true,
		"my_shop":               true,
		"-shop":                 true,
		"shop-":                 true,
		strings.Repeat("a", 58): false,
		strings.Repeat("a", 59): true, // <name>-prod would exceed the 63 character namespace limit
	} {
		t.Run(name, func(t *testing.T) {
			repo, err := ri.NewRepo(ri.KubeAppRepoFlag, filepath.Join(t.TempDir(), name), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.Init(); (err != nil) != wantErr {
				t.Errorf("Init error = %v, wantErr %v", err, wantErr)
			}
		})
	}
}

func TestFromYAMLFileContents(t *testing.T) {
	repo, err := ri.FromYAML([]byte(customYAML))
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Directories) != 2 {
		t.Fatalf("got %d dirs, want 2", len(repo.Directories))
	}
	files := repo.Directories[1].Files
	if len(files) != 1 || files[0].Name != "README.md" || files[0].Content != "# {{ .Name }}\n" {
		t.Errorf("unexpected files: %+v", files)
	}
}

// initFromYAML parses data as a repo template, names it name and runs Init.
func initFromYAML(t *testing.T, name, data string) error {
	t.Helper()
	repo, err := ri.FromYAML([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	repo.Name = name
	return repo.Init()
}

func TestNameIsBaseOfPath(t *testing.T) {
	const data = "dirs:\n  - path: \"./\"\n    files:\n      - name: name.txt\n        content: '{{ .Name }}'\n"

	tests := []struct {
		repoName string // as passed to -name, relative to the working directory
		want     string
	}{
		{"proj", "proj"},
		{"nested/dir/proj", "proj"},
		{"nested/dir/proj/", "proj"},
		{"./proj", "proj"},
	}
	for _, tc := range tests {
		t.Run(tc.repoName, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := initFromYAML(t, tc.repoName, data); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(tc.repoName, "name.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("{{ .Name }} = %q, want %q", got, tc.want)
			}
		})
	}

	// an empty name or "." creates the repo in the working directory, so Name is that directory's name
	for _, repoName := range []string{"", "."} {
		t.Run("cwd "+repoName, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "workdir")
			if err := os.Mkdir(dir, 0755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)
			if err := initFromYAML(t, repoName, data); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile("name.txt")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "workdir" {
				t.Errorf("{{ .Name }} = %q, want %q", got, "workdir")
			}
		})
	}
}

func TestRawFilesAndEscaping(t *testing.T) {
	const data = `
dirs:
  - path: "./"
    files:
      - name: raw.yml
        raw: true
        content: 'name: "{{ some_var }} {{ .Name }}"'
      - name: escaped.yml
        content: 'name: "{{ "{{" }} some_var }} {{ .Name }}"'
`
	t.Chdir(t.TempDir())
	if err := initFromYAML(t, "proj", data); err != nil {
		t.Fatal(err)
	}

	for file, want := range map[string]string{
		"proj/raw.yml":     `name: "{{ some_var }} {{ .Name }}"`,
		"proj/escaped.yml": `name: "{{ some_var }} proj"`,
	} {
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", file, got, want)
		}
	}
}

func TestRunCommandsTemplatesArgsAndRunsInRepoDir(t *testing.T) {
	if _, err := exec.LookPath("touch"); err != nil {
		t.Skip("touch not available")
	}

	repo, err := ri.FromYAML([]byte(`
commands:
  - name: touch
    args: ["{{ .Name }}.marker", "plain.marker"]
`))
	if err != nil {
		t.Fatal(err)
	}
	repo.Name = "nested/dir/proj"

	t.Chdir(t.TempDir())
	if err := os.MkdirAll(repo.Name, 0755); err != nil {
		t.Fatal(err)
	}
	if err := repo.RunCommands(); err != nil {
		t.Fatal(err)
	}

	// args are templated ({{ .Name }} -> proj) and the command ran inside the repo directory
	for _, f := range []string{"proj.marker", "plain.marker"} {
		if _, err := os.Stat(filepath.Join(repo.Name, f)); err != nil {
			t.Errorf("expected %s in the repo directory: %v", f, err)
		}
	}
}

func TestRunCommandsOptional(t *testing.T) {
	const missing = "ri-test-no-such-command"

	tests := []struct {
		name     string
		command  string
		optional bool
		wantErr  bool
	}{
		{"missing required", missing, false, true},
		{"missing optional", missing, true, false},
		{"failing required", "false", false, true},
		{"failing optional", "false", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := ri.Repo{Commands: []ri.Command{{Name: tc.command, Optional: tc.optional}}}
			err := repo.RunCommands()
			if (err != nil) != tc.wantErr {
				t.Errorf("RunCommands() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRunCommandsStopsOnRequiredFailure(t *testing.T) {
	if _, err := exec.LookPath("touch"); err != nil {
		t.Skip("touch not available")
	}

	repo := ri.Repo{Commands: []ri.Command{
		{Name: "false"},
		{Name: "touch", Args: []string{"should-not-exist"}},
	}}
	t.Chdir(t.TempDir())
	if err := repo.RunCommands(); err == nil {
		t.Fatal("RunCommands succeeded, want error")
	}
	if _, err := os.Stat("should-not-exist"); err == nil {
		t.Error("command after a failed required command still ran")
	}
}

func TestRunCommandsInvalidArgTemplate(t *testing.T) {
	for name, arg := range map[string]string{
		"parse error":   "{{ .Name",
		"unknown field": "{{ .Nope }}",
	} {
		t.Run(name, func(t *testing.T) {
			repo := ri.Repo{Name: "proj", Commands: []ri.Command{{Name: "true", Args: []string{arg}}}}
			if err := repo.RunCommands(); err == nil {
				t.Error("RunCommands succeeded, want template error")
			}
		})
	}
}

// The auto-merge workflow is full of GitHub Actions ${{ }} expressions; it must be written verbatim.
func TestGoTemplateAutomergeWorkflowIsRaw(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myrepo")
	repo, err := ri.NewRepo(ri.GoRepoFlag, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Init(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "dependabot-automerge.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"${{ secrets.GITHUB_TOKEN }}", "${{ github.event.pull_request.html_url }}"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("workflow lost %q: expressions were templated", want)
		}
	}
}
