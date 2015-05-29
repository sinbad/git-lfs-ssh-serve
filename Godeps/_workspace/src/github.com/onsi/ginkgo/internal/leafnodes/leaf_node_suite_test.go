package leafnodes_test

import (
	. "github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/onsi/gomega"

	"testing"
)

func TestLeafNode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LeafNode Suite")
}
