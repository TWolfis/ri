package ri

import (
	_ "embed"
)

//go:embed templates/ansible_repo.yaml
var ansibleRepoData []byte

type AnsibleRepo struct {
	Repo
}

func newAnsibleRepo(name string) IRI {
	repo, err := FromYAML(ansibleRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &AnsibleRepo{
		Repo: *repo,
	}
}
