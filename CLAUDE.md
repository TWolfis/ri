# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Ri (Repo init)

The goal of this project is to create a simple tool that initializes a new repository with a standard structure and configuration. It will help developers quickly set up new projects with best practices in mind.

### Features

- Initialize a new repository with a standard directory structure and CLAUDE.md file.
- Sane defaults for common project configurations.
- Extensible to support different programming languages and frameworks in the future.
- Let users define their own templates for project initialization.

## Commands

- Build: `make build` (outputs `bin/ri`)
- Run: `make run ARGS="-type <go|c|python|ansible|kube_app|terraform|packer|custom> -name <path> [-yaml file.yaml] [-skip-commands] [-force]"` (`-type` and `-name` are required)
- Test all: `make test`
- Test a single test: `go test -run TestName .`
- Vet: `make vet`; format: `make fmt`
- Full check (gofmt, vet, tests): `make check`
- Install to `$GOPATH/bin`: `make install`

## Architecture

- Module path is `github.com/TWolfis/ri`. The `ri` package (repo root) is the library; `cmd/ri` is a thin CLI over `ri.NewRepo`. `main` only calls `run(args, stdout, stderr) int`, which is what `cmd/ri/main_test.go` exercises; errors are printed as `ri: <err>` with exit code 1 (2 for bad usage), never panics.
- `ri.NewRepo(type, name, yamlFile, opts...)` returns `(IRI, error)`. Options (`WithOverwrite`) are applied through the unexported `IRI.repo()` method, which every repo type gets by embedding `Repo`.
- `Init` plans first (`Repo.plan`: validates `name_pattern`, renders every file and command arg, no filesystem access), then refuses with `ErrFilesExist` if any target file exists (unless `WithOverwrite`), then writes with `O_EXCL`. Template errors and conflicts therefore never leave a half-written repo. Keep this order when changing `Init`.
- A repo type is a YAML template (`Repo` in `repo_init.go`: `dirs` -> `files`, plus `commands`). Built-in templates live in `internal/templates/<type>_repo.yaml`, embedded by the `internal/templates` package (`//go:embed *.yaml`); `NewRepo` finds one by the flag's `String()` name, so there is no per-type Go code. `custom` loads a user-supplied YAML file instead.

## Adding a new repository type

There are two ways to add a repo type: a built-in one (compiled in, selected with `-type <name>`) or a user-supplied YAML file (`-type custom -yaml file.yaml`, no code changes). Both use the same YAML schema, described below.

To add a built-in type `foo` (no other Go code is needed; `TestTemplatesMatchRepoFlags` fails if a flag and a template do not pair up):

1. `internal/templates/foo_repo.yaml`: the template (schema below). Use `go_repo.yaml` as the reference. The file name must be exactly `<flag name>_repo.yaml`.
2. `repo_init_flag.go`: add `FooRepoFlag` to the const block **before** `CustomRepoFlag` (and keep `lastRepoFlag` last, since `RepoFlags()` iterates up to it), then add `"foo"` cases to both `String()` and `Set()`.
3. `repo_init_test.go`: add an entry for the flag to `repoCases` with the `wantPaths` (and `wantContents` for rendered files) it must create. `TestNewRepoInit` and `TestRepoFlags` fail until this exists.
4. `go test ./...`, then generate it for real: `go run ./cmd/ri -type foo -name /tmp/foo-proj` and build/run the output. The `-type` help text lists new flags automatically.

### Template schema

```yaml
---
name: foo_repo          # overwritten by -name at runtime
name_pattern: '^[a-z]+$' # optional: regexp the repo name (last element of -name) must match
dirs:                   # created under the -name path, in order
  - path: "src"         # directory only
  - path: "./"          # "./" is the repo root
    files:
      - name: "README.md"
        content: |
          # {{ .Name }}
      - name: "ci.yml"
        raw: true       # write content verbatim (no templating)
        content: |
          run: echo ${{ github.sha }}
commands:               # run after the files exist, inside the repo directory
  - name: git
    args: ["init"]
    optional: true      # warn instead of failing if missing or failing
```

Use `name_pattern` when generated files break on unusual names (`kube_app` uses it for DNS-safe Kubernetes names).

Conventions for built-in templates: include `README.md`, `CLAUDE.md` (with the project's real commands and layout), a `.gitignore`, and `git init` as an optional command. Put network- or tool-dependent commands (`uv sync`, `ansible-galaxy ...`) behind `optional: true`. Empty directories need a `.gitkeep` file to survive in git. Directory paths and file names are static; only file `content` and command `args` are templated.

## Templating

- File `content` and command `args` are executed as `text/template`; the YAML itself is not. Directory paths and file names are not templated.
- Template data: `{{ .Name }}` is the base of the `-name` path (an empty name or `.` resolves to the working directory's name). Unknown fields are errors (`missingkey=error`).
- Files that contain literal `{{ }}` (Jinja, GitHub Actions) set `raw: true`. In a templated file, write `{{ "{{" }}` for a literal `{{`.
- Everything is created under the `-name` path (as given, not just its base).
- `commands:` run after `Init`, from inside the repo directory, via `IRI.RunCommands` (called by `cmd/ri` unless `-skip-commands`). Each command's `args` are templated. A command with `optional: true` (all built-in defaults: `git init`, `go mod tidy`, `uv sync`, `ansible-galaxy ...`) only warns if the tool is missing or fails; otherwise a failure stops the run.
- Library tests call `Init` only, and CLI tests pass `-skip-commands`, so tests never shell out to git/uv/go; command behavior is tested separately with `touch`/`false`.
