package ri

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v2"
)

// ErrFilesExist is returned (wrapped) by Init when files it would create already exist.
var ErrFilesExist = errors.New("files already exist")

type IRI interface {
	// Init creates the directories and files specified in the Repo struct.
	// It renders everything first and refuses (ErrFilesExist) to overwrite existing files unless
	// WithOverwrite was given, so a failure leaves nothing half written.
	Init() error

	// RunCommands runs the template's commands inside the created repository. Call it after Init.
	RunCommands() error

	// repo gives NewRepo access to the underlying Repo to apply Options.
	repo() *Repo
}

// Option customizes a repository created by NewRepo.
type Option func(*Repo)

// WithOverwrite lets Init replace files that already exist.
func WithOverwrite() Option {
	return func(r *Repo) { r.overwrite = true }
}

// NewRepo creates a new repository of the specified type (Go, C, Python, Ansible, KubeApp, Terraform, Packer or Custom) with the given name and optional YAML file for custom repositories.
func NewRepo(repoType RepoFlag, name string, yamlFile *string, opts ...Option) (IRI, error) {
	var repo IRI
	switch repoType {
	case GoRepoFlag:
		repo = newGoRepo(name)
	case CRepoFlag:
		repo = newCRepo(name)
	case PythonRepoFlag:
		repo = newPythonRepo(name)
	case AnsibleRepoFlag:
		repo = newAnsibleRepo(name)
	case KubeAppRepoFlag:
		repo = newKubeAppRepo(name)
	case TerraformRepoFlag:
		repo = newTerraformRepo(name)
	case PackerRepoFlag:
		repo = newPackerRepo(name)
	case CustomRepoFlag:
		if yamlFile == nil || *yamlFile == "" {
			return nil, errors.New("a YAML file is required for the custom repo type")
		}
		var err error
		if repo, err = newCustomRepo(name, *yamlFile); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid repo type %d", int(repoType))
	}

	for _, opt := range opts {
		opt(repo.repo())
	}
	return repo, nil
}

type Repo struct {
	// Name is the name of the repository.
	Name string `yaml:"name"`

	// NamePattern is an optional regular expression the repository name (the last element of the
	// path given to -name) must match, for templates whose files break on unusual names.
	NamePattern string `yaml:"name_pattern"`

	// Directories is a list of directories to create in the repository.
	Directories []Directory `yaml:"dirs"`

	// Commands is a list of commands to run. Their args are templated like file content.
	Commands []Command `yaml:"commands"`

	// overwrite lets Init replace existing files.
	overwrite bool
}

func (r *Repo) repo() *Repo { return r }

// Directory is a directory to create, together with the files that go in it.
type Directory struct {
	// Path is the path of the directory to create.
	Path string `yaml:"path"`

	// Files is a list of files to create in the directory.
	Files []File `yaml:"files"`
}

// File is a file to create.
type File struct {
	// Name is the path of the file to create.
	Name string `yaml:"name"`

	// Content is the content of the file to create. It is executed as a text/template unless Raw is set.
	Content string `yaml:"content"`

	// Raw writes Content exactly as written, so it may contain literal {{ }} (Jinja, GitHub Actions, ...).
	// Without Raw, a literal {{ can still be produced with {{ "{{" }}.
	Raw bool `yaml:"raw"`
}

// Command is a command to run.
type Command struct {
	// Name is the name of the command to run.
	Name string `yaml:"name"`

	// Args is the list of arguments to pass to the command.
	Args []string `yaml:"args"`

	// Optional makes a missing or failing command print a warning instead of stopping.
	// Use it for tools that may not be installed or that need network access.
	Optional bool `yaml:"optional"`
}

// templateData is the data available to templates, e.g. {{ .Name }}.
type templateData struct {
	// Name is the name of the repository: the last element of the path given as the repo name.
	Name string
}

// templateData builds the data passed to templates. The repo name may be a path such as
// "projects/foo" or "."; templates get its base ("foo", or the current directory's name).
func (r *Repo) templateData() (templateData, error) {
	abs, err := filepath.Abs(r.Name)
	if err != nil {
		return templateData{}, fmt.Errorf("resolving repo name %q: %w", r.Name, err)
	}
	return templateData{Name: filepath.Base(abs)}, nil
}

// render executes text as a text/template using the repo's template data.
// Only file content and command args are templated; the YAML itself is parsed as-is.
func (r *Repo) render(label, text string) (string, error) {
	data, err := r.templateData()
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(label).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing template for %s: %w", label, err)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("rendering template for %s: %w", label, err)
	}
	return sb.String(), nil
}

// plannedFile is a file that Init is about to write, with its content already rendered.
type plannedFile struct {
	path    string
	content string
}

// plan renders the whole repository without touching the filesystem, so name and template
// errors are reported before anything is written.
func (r *Repo) plan() (dirs []string, files []plannedFile, err error) {
	if err := r.checkName(); err != nil {
		return nil, nil, err
	}

	for _, dir := range r.Directories {
		// everything is created under a root directory named after the repo
		dirPath := filepath.Join(r.Name, dir.Path)
		dirs = append(dirs, dirPath)

		for _, file := range dir.Files {
			content := file.Content
			if !file.Raw {
				if content, err = r.render(file.Name, file.Content); err != nil {
					return nil, nil, err
				}
			}
			files = append(files, plannedFile{path: filepath.Join(dirPath, file.Name), content: content})
		}
	}

	// commands run later, but report broken arg templates now
	for _, cmd := range r.Commands {
		if _, err := r.commandArgs(cmd); err != nil {
			return nil, nil, err
		}
	}
	return dirs, files, nil
}

// checkName validates the repository name against NamePattern.
func (r *Repo) checkName() error {
	if r.NamePattern == "" {
		return nil
	}

	re, err := regexp.Compile(r.NamePattern)
	if err != nil {
		return fmt.Errorf("invalid name_pattern %q: %w", r.NamePattern, err)
	}
	data, err := r.templateData()
	if err != nil {
		return err
	}
	if !re.MatchString(data.Name) {
		return fmt.Errorf("repository name %q is not valid for this template: it must match %s", data.Name, r.NamePattern)
	}
	return nil
}

func (r *Repo) Init() error {
	dirs, files, err := r.plan()
	if err != nil {
		return err
	}

	if !r.overwrite {
		var existing []string
		for _, f := range files {
			if _, err := os.Lstat(f.path); err == nil {
				existing = append(existing, f.path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if len(existing) > 0 {
			return fmt.Errorf("%w, refusing to overwrite (use -force to replace them): %s", ErrFilesExist, strings.Join(existing, ", "))
		}
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if r.overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
			return err
		}
		if err := writeFile(f.path, f.content, flags); err != nil {
			return err
		}
	}
	return nil
}

// writeFile writes content to path, reporting close errors too. With O_EXCL in flags it also
// fails if the file appeared since the existence check.
func writeFile(path, content string, flags int) error {
	out, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return err
	}
	if _, err := out.WriteString(content); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// commandArgs renders the args of cmd.
func (r *Repo) commandArgs(cmd Command) ([]string, error) {
	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		rendered, err := r.render(fmt.Sprintf("%s arg %d", cmd.Name, i+1), arg)
		if err != nil {
			return nil, err
		}
		args[i] = rendered
	}
	return args, nil
}

// RunCommands runs the repo's commands in order, from inside the repo directory.
func (r *Repo) RunCommands() error {
	for _, cmd := range r.Commands {
		args, err := r.commandArgs(cmd)
		if err != nil {
			return err
		}

		if err := runCommand(r.Name, cmd.Name, args...); err != nil {
			if cmd.Optional {
				fmt.Fprintf(os.Stderr, "warning: skipping optional command: %v\n", err)
				continue
			}
			return err
		}
	}
	return nil
}

// runCommand runs name with args in dir ("" means the current directory).
func runCommand(dir, name string, args ...string) error {
	// test if the command exists in PATH
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("command %s not found in PATH: %w", name, err)
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// FromYAML parses the given YAML data into a Repo struct.
func FromYAML(data []byte) (*Repo, error) {

	// unmarshal the YAML data into a Repo struct
	var repo Repo
	err := yaml.Unmarshal(data, &repo)
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

func FromYamlFile(path string) (*Repo, error) {
	// read the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return FromYAML(data)
}
