# ri

`ri` (repo init) creates a new repository with a sensible structure, starter files and a `CLAUDE.md`, then runs the usual setup commands (`git init`, `go mod tidy`, `uv sync`, ...). Templates are plain YAML, so you can add your own.

## Install

```sh
go install github.com/TWolfis/ri/cmd/ri@latest
```

## Usage

```sh
ri -type go -name my-service          # ./my-service, module "my-service"
ri -type python -name ~/code/my-tool  # the name used in files is the last path element: "my-tool"
ri -type terraform -name . -force     # in the current directory, replacing files that exist
ri -type custom -yaml mytemplate.yaml -name my-repo
ri -list-types
```

| Flag | Description |
|---|---|
| `-type` | Repository type (required, see below). |
| `-name` | Path of the repository to create (required). Use `.` for the current directory. |
| `-yaml` | Template file for `-type custom`. |
| `-skip-commands` | Only create files; do not run the template's commands. |
| `-force` | Overwrite files that already exist. Without it `ri` refuses, and writes nothing. |
| `-list-types` | Print the supported types. |

## Types

| Type | Creates | Runs |
|---|---|---|
| `go` | `cmd/app`, `internal`, `pkg`, `go.mod`, Makefile | `git init`, `go mod tidy` |
| `c` | `src`, `include`, Makefile with dependency tracking, `.clang-format` | `git init` |
| `python` | `src/app`, `tests`, `pyproject.toml` (uv, pytest, ruff) | `git init`, `uv sync` |
| `ansible` | inventory, playbook, `common` role, `ansible.cfg` | `git init`, `ansible-galaxy collection install` |
| `kube_app` | Kustomize `base` and `overlays/{dev,prod}`, Argo CD `AppProject` and `Application`s | `git init` |
| `terraform` | root module, `modules/`, per-environment `envs/*.tfvars` | `git init`, `terraform init` |
| `packer` | HCL2 template with the Docker builder, `scripts/` | `git init`, `packer init .` |

Every type also gets a `README.md`, a `CLAUDE.md` describing its commands and layout, and a `.gitignore`.

Commands that need a tool or network access are optional: if `uv`, `terraform` etc. are missing, `ri` prints a warning and carries on. Use `-skip-commands` to skip them entirely.

`kube_app` requires a lowercase DNS-style name (letters, digits and `-`, at most 58 characters), because the name becomes Kubernetes namespaces. Generated Argo CD manifests contain a `TODO` for your repository URL.

## Custom templates

A template is a YAML file describing directories, files and commands:

```yaml
name: my_template          # overwritten by -name
name_pattern: '^[a-z-]+$'  # optional: the repository name must match this regular expression
dirs:                      # created under the -name path
  - path: "src"
  - path: "./"             # "./" is the repository root
    files:
      - name: README.md
        content: |
          # {{ .Name }}
      - name: ci.yml
        raw: true          # written exactly as is, so {{ }} needs no escaping
        content: |
          run: echo ${{ github.sha }}
commands:                  # run inside the repository after the files are written
  - name: git
    args: ["init"]
    optional: true         # warn instead of failing if missing or failing
```

File `content` and command `args` are [Go templates](https://pkg.go.dev/text/template). `{{ .Name }}` is the last element of the `-name` path. Unknown fields are errors. In a templated file, write `{{ "{{" }}` for a literal `{{`, or set `raw: true` on the file. Directory paths and file names are not templated.

## Development

```sh
make check   # gofmt, go vet and tests
make build   # bin/ri
```

See [CLAUDE.md](CLAUDE.md) for the architecture and how to add a built-in type.

## License

[MIT](LICENSE)
