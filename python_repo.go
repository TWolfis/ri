package ri

import (
	_ "embed"
)

//go:embed templates/python_repo.yaml
var pythonRepoData []byte

type PythonRepo struct {
	Repo
}

func newPythonRepo(name string) IRI {
	repo, err := FromYAML(pythonRepoData)
	if err != nil {
		panic(err)
	}

	repo.Name = name
	return &PythonRepo{
		Repo: *repo,
	}
}
