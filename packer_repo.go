package ri

import (
	_ "embed"
)

//go:embed templates/packer_repo.yaml
var packerRepoData []byte

type PackerRepo struct {
	Repo
}

func newPackerRepo(name string) IRI {
	repo, err := FromYAML(packerRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &PackerRepo{
		Repo: *repo,
	}
}
