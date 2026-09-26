package ri

import (
	_ "embed"
)

//go:embed templates/kube_app_repo.yaml
var kubeAppRepoData []byte

type KubeAppRepo struct {
	Repo
}

func newKubeAppRepo(name string) IRI {
	repo, err := FromYAML(kubeAppRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &KubeAppRepo{
		Repo: *repo,
	}
}
