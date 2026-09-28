package ri_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TWolfis/ri"
)

// Create a Go repository and read back the go.mod it generated. The repository's name is the last
// element of the path, which is also what {{ .Name }} renders to in templates.
func ExampleNewRepo() {
	dir, err := os.MkdirTemp("", "ri-example")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)

	repo, err := ri.NewRepo(ri.GoRepoFlag, filepath.Join(dir, "hello"), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := repo.Init(); err != nil {
		fmt.Println(err)
		return
	}

	gomod, err := os.ReadFile(filepath.Join(dir, "hello", "go.mod"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(string(gomod))
	// Output:
	// module hello
	//
	// go 1.27
}
