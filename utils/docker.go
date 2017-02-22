package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Push a container to a registry. This command does not login to the registry
// that will be pushed to.
//
// container -- Name of the container to be pushed
func Push(container string) error {
	var err error

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker command: %s", err)
	}

	pushCmdArgs := []string{"push", container}
	pushCmd := exec.Command(dockerCmd, pushCmdArgs...)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr

	err = pushCmd.Run()
	if err != nil {
		return fmt.Errorf("An error occurred pushing the container to ECR: %s", err)
	}

	return nil
}

// Tag a container with the new tag provided. This method will also pull the
// old container name provided if it is not currently cached locally.
//
// old -- Name of the old container to use
// new -- Name of the new container to push
// region -- AWS region to use
// profile -- AWS profile name to use
func TagContainer(old, new, region, profile string) error {
	var err error
	tagCmdArgs := []string{"tag", old, new}

	if !localContainerFound(old) {
		if err := Pull(old); err != nil {
			return err
		}
	}

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker command: %s", err)
	}

	dockerTagCmd := exec.Command(dockerCmd, tagCmdArgs...)
	dockerTagCmd.Stdout = os.Stdout
	dockerTagCmd.Stderr = os.Stderr

	err = dockerTagCmd.Run()
	if err != nil {
		return fmt.Errorf("Error tagging container with build date and repo: %s", err)
	}

	return nil
}

// Pull the provided container name from a registry. You must login to the registry
// before using this command.
//
// container -- Name of the container to pull from a remote repository
func Pull(container string) error {
	var err error

	fmt.Printf("Attempting to pull container: %s", container)
	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker command: %s", err)
	}

	pushCmdArgs := []string{"pull", container}
	pushCmd := exec.Command(dockerCmd, pushCmdArgs...)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr

	err = pushCmd.Run()
	if err != nil {
		return fmt.Errorf("An error occurred pulling the container: %s", err)
	}

	return nil
}

// Deletes a set of containers based on a given label of a <key>=<value> pair
//
// label -- The label to lookup for containers to delete
func CleanUpByLabel(label string) error {
	var output bytes.Buffer

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker command: %s", err)
	}

	findArgs := []string{"images", "-f", label, "--format={{.ID}}"}
	findCmd := exec.Command(dockerCmd, findArgs...)
	findCmd.Stdout = &output

	err = findCmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to find images using label '%s': %s", label, err)
	}

	ids := []string{}
	for _, id := range strings.Split(output.String(), "\n") {
		if len(ids) == 0 {
			ids = append(ids, id)
			continue
		}
		if id == "" {
			continue
		}

		exists := false
		for _, v := range ids {
			if v == id {
				exists = true
			}
		}
		if !exists {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return fmt.Errorf("No images found")
	}

	rmiArgs := []string{"rmi", "-f"}
	rmiArgs = append(rmiArgs, ids...)

	rmiCmd := exec.Command(dockerCmd, rmiArgs...)
	rmiCmd.Stdout = os.Stdout
	rmiCmd.Stderr = os.Stderr

	err = rmiCmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to delete images: %s", err)
	}

	return nil
}

// Looks up a label for a container to determine the build date of the container.
//
// name -- Name of the container
// tag -- Tag for the container
// label -- Name of the label that contains the build date
func GetContainerBuildDate(name, tag, label string) (string, error) {
	var out bytes.Buffer

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return "", err
	}
	cmdArgs := []string{"inspect"}
	cmdArgs = append(cmdArgs, fmt.Sprintf("--format='{{index .ContainerConfig.Labels \"%s\"}}'", label))
	cmdArgs = append(cmdArgs, fmt.Sprintf("%s:%s", name, tag))

	cmd := exec.Command(dockerCmd, cmdArgs...)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func localContainerFound(container string) bool {
	var err error
	var out bytes.Buffer

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return false
	}

	imagesArgs := []string{"images", "--format={{.Repository}}:{{.Tag}}"}

	cmd := exec.Command(dockerCmd, imagesArgs...)
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return false
	}

	for _, image := range strings.Split(out.String(), "\n") {
		if strings.Compare(strings.TrimSpace(image), strings.TrimSpace(container)) == 0 {
			return true
		}
	}
	return false
}
