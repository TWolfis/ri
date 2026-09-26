package ri

import "fmt"

type CustomRepo struct {
	Repo
}

// newCustomRepo creates a new custom repository with the given name from a user supplied YAML file.
func newCustomRepo(name string, yamlFile string) (IRI, error) {
	repo, err := FromYamlFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", yamlFile, err)
	}

	repo.Name = name
	return &CustomRepo{
		Repo: *repo,
	}, nil
}
