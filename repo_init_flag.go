package ri

import (
	"fmt"
	"iter"
)

// flag for creating a repo

type RepoFlag int

const (
	// GoRepo is the flag for creating a Go repository.
	GoRepoFlag RepoFlag = iota
	// CRepoFlag is the flag for creating a C repository.
	CRepoFlag
	// PythonRepoFlag is the flag for creating a Python repository.
	PythonRepoFlag
	// AnsibleRepoFlag is the flag for creating an Ansible repository.
	AnsibleRepoFlag
	// KubeAppRepoFlag is the flag for creating a Kubernetes application repository (Kustomize + Argo CD).
	KubeAppRepoFlag
	// TerraformRepoFlag is the flag for creating a Terraform repository.
	TerraformRepoFlag
	// PackerRepoFlag is the flag for creating a Packer repository.
	PackerRepoFlag
	// CustomRepoFlag is the flag for creating a repository from a user supplied YAML file.
	CustomRepoFlag
	lastRepoFlag
)

func (f RepoFlag) String() string {
	switch f {
	case GoRepoFlag:
		return "go"
	case CRepoFlag:
		return "c"
	case PythonRepoFlag:
		return "python"
	case AnsibleRepoFlag:
		return "ansible"
	case KubeAppRepoFlag:
		return "kube_app"
	case TerraformRepoFlag:
		return "terraform"
	case PackerRepoFlag:
		return "packer"
	case CustomRepoFlag:
		return "custom"
	default:
		return "unknown"
	}
}

func (f *RepoFlag) Set(value string) error {
	switch value {
	case "go":
		*f = GoRepoFlag
	case "c":
		*f = CRepoFlag
	case "python":
		*f = PythonRepoFlag
	case "ansible":
		*f = AnsibleRepoFlag
	case "kube_app":
		*f = KubeAppRepoFlag
	case "terraform":
		*f = TerraformRepoFlag
	case "packer":
		*f = PackerRepoFlag
	case "custom":
		*f = CustomRepoFlag
	default:
		return fmt.Errorf("invalid repo flag: %s", value)
	}
	return nil
}

func (f *RepoFlag) Type() string {
	return "repo"
}

func RepoFlags() iter.Seq[RepoFlag] {
	return func(yield func(RepoFlag) bool) {
		for f := RepoFlag(0); f < lastRepoFlag; f++ {
			if !yield(f) {
				return
			}
		}
	}
}
