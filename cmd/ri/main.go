// Command ri initializes a new repository from a template.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TWolfis/ri"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main without the process globals, so it can be tested. It returns the exit code.
func run(args []string, stdout, stderr io.Writer) int {
	var (
		repoType     ri.RepoFlag
		name         string
		yamlFile     string
		skipCommands bool
		force        bool
		listTypes    bool
	)

	fs := flag.NewFlagSet("ri", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: ri -type <type> -name <path> [flags]\n\nCreates a repository in <path> from a template. Types: %s\n\n", typeNames())
		fs.PrintDefaults()
	}
	fs.Var(&repoType, "type", fmt.Sprintf("Type of repository to create (%s)", typeNames()))
	fs.StringVar(&name, "name", "", "Path of the repository to create; its last element is the name used in templates (use . for the current directory)")
	fs.StringVar(&yamlFile, "yaml", "", "Path to the YAML template for -type custom")
	fs.BoolVar(&skipCommands, "skip-commands", false, "Only create files; do not run the template's commands (git init, uv sync, ...)")
	fs.BoolVar(&force, "force", false, "Overwrite files that already exist")
	fs.BoolVar(&listTypes, "list-types", false, "List all supported repository types")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2 // the flag package already printed the error and usage
	}

	if listTypes {
		fmt.Fprintln(stdout, typeNames())
		return 0
	}

	if fs.NArg() > 0 {
		return usageError(fs, stderr, "unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["type"] {
		return usageError(fs, stderr, "-type is required (%s)", typeNames())
	}
	if name == "" {
		return usageError(fs, stderr, "-name is required (use -name . for the current directory)")
	}

	var opts []ri.Option
	if force {
		opts = append(opts, ri.WithOverwrite())
	}

	repo, err := ri.NewRepo(repoType, name, &yamlFile, opts...)
	if err != nil {
		fmt.Fprintf(stderr, "ri: %v\n", err)
		return 1
	}
	if err := repo.Init(); err != nil {
		fmt.Fprintf(stderr, "ri: %v\n", err)
		return 1
	}

	if !skipCommands {
		if err := repo.RunCommands(); err != nil {
			fmt.Fprintf(stderr, "ri: %v\n", err)
			return 1
		}
	}
	return 0
}

// usageError prints a message and the usage text, and returns the exit code for bad usage.
func usageError(fs *flag.FlagSet, stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "ri: "+format+"\n\n", args...)
	fs.Usage()
	return 2
}

// typeNames lists every supported repository type for help text.
func typeNames() string {
	var names []string
	for f := range ri.RepoFlags() {
		names = append(names, f.String())
	}
	return strings.Join(names, ", ")
}
