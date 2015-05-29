package main

import (
	. "github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/onsi/gomega"

	"testing"
)

func TestGitLfsSshServe(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "git-lfs ssh server suite")
}
