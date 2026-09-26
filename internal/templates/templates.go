// Package templates embeds the built-in repository templates.
package templates

import (
	"embed"
	"fmt"
	"strings"
)

const suffix = "_repo.yaml"

//go:embed *.yaml
var files embed.FS

// Get returns the YAML of the built-in template for a repository type such as "go" or "kube_app".
func Get(repoType string) ([]byte, error) {
	data, err := files.ReadFile(repoType + suffix)
	if err != nil {
		return nil, fmt.Errorf("no built-in template for type %q", repoType)
	}
	return data, nil
}

// Types returns the repository types that have a built-in template.
func Types() []string {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil // cannot happen: the pattern above matched at build time
	}

	var types []string
	for _, e := range entries {
		types = append(types, strings.TrimSuffix(e.Name(), suffix))
	}
	return types
}
