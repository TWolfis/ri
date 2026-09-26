package ri

import (
	_ "embed"
)

//go:embed templates/go_repo.yaml
var goRepoData []byte

type GoRepo struct {
	Repo
}

func newGoRepo(name string) IRI {
	repo, err := FromYAML(goRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &GoRepo{
		Repo: *repo,
	}
}
