package main

import (
	"flag"
	"fmt"
	"strings"
	"text/template"
)

// completionShells lists the shells -completion can generate a script for.
var completionShells = []string{"bash", "zsh"}

// What a flag's value completes to. Flags not listed here complete as plain file names,
// except boolean flags, which take no value.
const (
	valueNone   = ""      // boolean flag: no value
	valueTypes  = "types" // one of the repo types
	valueDir    = "dir"
	valueYAML   = "yaml"
	valueShells = "shells"
)

// flagValueKinds gives the value completion for flags that need something better than file names.
var flagValueKinds = map[string]string{
	"type":       valueTypes,
	"name":       valueDir,
	"yaml":       valueYAML,
	"completion": valueShells,
}

type completionFlag struct {
	Name  string
	Usage string
	Kind  string // one of the value* constants
}

type completionData struct {
	Flags  []completionFlag
	Types  string // space separated
	Shells string // space separated
}

// completionScript returns a completion script for shell, generated from the flags defined on fs
// and the given repo types, so it cannot drift from the real command line.
func completionScript(shell string, fs *flag.FlagSet, types []string) (string, error) {
	tmpl, ok := map[string]*template.Template{"bash": bashTemplate, "zsh": zshTemplate}[shell]
	if !ok {
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(completionShells, ", "))
	}

	data := completionData{Types: strings.Join(types, " "), Shells: strings.Join(completionShells, " ")}
	fs.VisitAll(func(f *flag.Flag) { // visits in lexical order
		kind := flagValueKinds[f.Name]
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			kind = valueNone
		} else if kind == "" {
			kind = "file"
		}
		data.Flags = append(data.Flags, completionFlag{Name: f.Name, Usage: f.Usage, Kind: kind})
	})

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// zshEscape makes s safe inside a single quoted _arguments spec's [description].
func zshEscape(s string) string {
	return strings.NewReplacer(`'`, `'\''`, `[`, `\[`, `]`, `\]`, `:`, `\:`).Replace(s)
}

var bashTemplate = template.Must(template.New("bash").Parse(`# bash completion for ri
# Load it with: eval "$(ri -completion bash)"
# ("source <(...)" does not work in the bash 3.2 that ships with macOS.)

# _ri_paths <dirs|yaml|files> <word>: fill COMPREPLY with matching paths.
# Written without extglob or word splitting so it works in any shell configuration.
_ri_paths() {
    local mode="$1" word="$2" f
    type compopt >/dev/null 2>&1 && compopt -o filenames
    COMPREPLY=()
    while IFS= read -r f; do
        case "$mode" in
            dirs) [ -d "$f" ] || continue ;;
            yaml) [ -d "$f" ] || case "$f" in *.yaml|*.yml) ;; *) continue ;; esac ;;
        esac
        COMPREPLY+=("$f")
    done < <(compgen -f -- "$word")
}

_ri() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    COMPREPLY=()

    case "$prev" in
{{- range .Flags}}{{if eq .Kind "types"}}
        -{{.Name}}|--{{.Name}})
            COMPREPLY=( $(compgen -W "{{$.Types}}" -- "$cur") )
            return 0 ;;
{{- else if eq .Kind "shells"}}
        -{{.Name}}|--{{.Name}})
            COMPREPLY=( $(compgen -W "{{$.Shells}}" -- "$cur") )
            return 0 ;;
{{- else if eq .Kind "dir"}}
        -{{.Name}}|--{{.Name}})
            _ri_paths dirs "$cur"
            return 0 ;;
{{- else if eq .Kind "yaml"}}
        -{{.Name}}|--{{.Name}})
            _ri_paths yaml "$cur"
            return 0 ;;
{{- else if eq .Kind "file"}}
        -{{.Name}}|--{{.Name}})
            _ri_paths files "$cur"
            return 0 ;;
{{- end}}{{end}}
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "{{range .Flags}}-{{.Name}} {{end}}" -- "$cur") )
    fi
    return 0
}

complete -F _ri ri
`))

var zshTemplate = template.Must(template.New("zsh").Funcs(template.FuncMap{"esc": zshEscape}).Parse(`#compdef ri
# zsh completion for ri
# Load it with: source <(ri -completion zsh)   (after compinit)
# or install it: ri -completion zsh > "${fpath[1]}/_ri"

_ri() {
    _arguments -s \
{{- range .Flags}}
{{- if eq .Kind "types"}}
        '-{{.Name}}[{{esc .Usage}}]:type:({{$.Types}})' \
{{- else if eq .Kind "shells"}}
        '-{{.Name}}[{{esc .Usage}}]:shell:({{$.Shells}})' \
{{- else if eq .Kind "dir"}}
        '-{{.Name}}[{{esc .Usage}}]:path:_files -/' \
{{- else if eq .Kind "yaml"}}
        '-{{.Name}}[{{esc .Usage}}]:template file:_path_files -g "*.(yaml|yml)" -/' \
{{- else if eq .Kind "file"}}
        '-{{.Name}}[{{esc .Usage}}]:file:_files' \
{{- else}}
        '-{{.Name}}[{{esc .Usage}}]' \
{{- end}}
{{- end}}
        && return 0
    return 1
}

if [ "$funcstack[1]" = "_ri" ]; then
    _ri "$@"
else
    compdef _ri ri
fi
`))
