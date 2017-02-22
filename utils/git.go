package utils

import (
	"bytes"
	"os/exec"
	"strings"
)

// Find the git SHA for a given tag
//
// tag -- Git tag to look up
func GitSHA(tag string) (string, error) {
	var out bytes.Buffer
	git, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(git, "rev-parse", "--short", tag)
	cmd.Stdout = &out

	err = cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// Finds the top-level directory in a git repository.
func GitToplevel() (string, error) {
	var out bytes.Buffer
	git, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(git, "rev-parse", "--show-toplevel")
	cmd.Stdout = &out

	err = cmd.Run()
	return strings.TrimSpace(out.String()), err
}
