package ri

import (
	_ "embed"
)

//go:embed templates/terraform_repo.yaml
var terraformRepoData []byte

type TerraformRepo struct {
	Repo
}

func newTerraformRepo(name string) IRI {
	repo, err := FromYAML(terraformRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &TerraformRepo{
		Repo: *repo,
	}
}
