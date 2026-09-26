package ri

import (
	_ "embed"
)

//go:embed templates/c_repo.yaml
var cRepoData []byte

type CRepo struct {
	Repo
}

func newCRepo(name string) IRI {
	repo, err := FromYAML(cRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &CRepo{
		Repo: *repo,
	}
}
