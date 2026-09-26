package templates_test

import (
	"slices"
	"testing"

	"github.com/TWolfis/ri"
	"github.com/TWolfis/ri/internal/templates"
)

// Every built-in RepoFlag needs a template named after it, and every template needs a flag.
func TestTemplatesMatchRepoFlags(t *testing.T) {
	var want []string
	for f := range ri.RepoFlags() {
		if f != ri.CustomRepoFlag { // custom templates come from the user, not from this package
			want = append(want, f.String())
		}
	}

	got := templates.Types()
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("templates = %v, RepoFlags without custom = %v", got, want)
	}
}

func TestGet(t *testing.T) {
	data, err := templates.Get("go")
	if err != nil || len(data) == 0 {
		t.Errorf("Get(go) = %d bytes, %v", len(data), err)
	}

	for _, name := range []string{"", "nope", "custom", "../go"} {
		if _, err := templates.Get(name); err == nil {
			t.Errorf("Get(%q) succeeded, want error", name)
		}
	}
}
