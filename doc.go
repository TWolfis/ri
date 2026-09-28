// Package ri (repo init) creates a new repository from a template: directories, starter files and
// the setup commands that normally follow (git init, go mod tidy, uv sync, ...).
//
// Templates are YAML. The built-in ones (go, c, python, ansible, kube_app, terraform and packer) are
// embedded in the module; a custom template can be loaded from any YAML file. File contents and
// command arguments are Go templates, so a template can use {{ .Name }} for the repository's name.
//
// Create a repository with [NewRepo], write its files with [IRI.Init], then run its setup commands
// with [IRI.RunCommands]:
//
//	repo, err := ri.NewRepo(ri.GoRepoFlag, "my-service", nil)
//	if err != nil {
//		return err
//	}
//	if err := repo.Init(); err != nil {
//		return err // includes ErrFilesExist if it would overwrite something
//	}
//	return repo.RunCommands()
//
// Init renders every file before writing anything and refuses to overwrite existing files unless
// [WithOverwrite] is given, so a failure never leaves a half-written repository.
//
// The command-line tool is in the cmd/ri directory: go install github.com/TWolfis/ri/cmd/ri@latest.
package ri
