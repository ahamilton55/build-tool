package cmd

import (
	"build_tool/utils"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	label string
	local bool
)

func init() {
	cleanupCli.Flags().StringVarP(&label, "label", "l", "", "<key>=<value> representation of a label")
	cleanupCli.Flags().StringVar(&containerName, "container", "", "Finds all tags related to the given container")
	cleanupCli.Flags().BoolVar(&local, "local", false, "Cleans up a local container for the environment")
	RootCmd.AddCommand(cleanupCli)
}

var cleanupCli = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleans up a set of Docker containers created during a build",
	Long:  `Cleans up a set of Docker containers`,
	Run: func(cmd *cobra.Command, args []string) {
		cleanupContainers()
	},
}

func cleanupContainers() {
	var err error

	if Config.Name == "" {
		utils.ErrorAndQuit("Container name not found in config", nil, 2)
	} else if Config.EcrRepo == "" {
		utils.ErrorAndQuit("ECR Repo not found in config", nil, 2)
	}

	if local {
		err = cleanUpLocalBuild()
	} else if containerName != "" {
		err = cleanUpUsingName(containerName)
	} else {
		err = cleanUpUsingLabel(label)
	}

	if err != nil {
		utils.ErrorAndQuit("", err, 3)
	} else {
		logger.Info("Completed successfully")
	}
}

func cleanUpLocalBuild() error {
	logger.Debug("Looking up job tag for container")
	dockerTag := utils.GetDockerJobTag()

	logger.Debug("Setup name for container")
	containerName = fmt.Sprintf("%s:%s", Config.Name, dockerTag)

	return cleanUpUsingName(containerName)
}

func cleanUpUsingLabel(label string) error {
	var err error

	if label == "" {
		label, err = utils.GetCommitLabel()
		if err != nil {
			return fmt.Errorf("Unable to build commit label: %s", err.Error())
		}
	}
	filterLabel := fmt.Sprintf("label=%s", label)
	log.Println(filterLabel)

	if err := utils.CleanUpByLabel(filterLabel); err != nil {
		return fmt.Errorf("Failed to clean up built containers: %s", err.Error())
	}

	return nil
}

func cleanUpUsingName(container string) error {
	var (
		containerNamesArgs = []string{"inspect", "--format='{{ .RepoTags }}'"}
		output             bytes.Buffer
		containerNames     []string
	)

	containerNamesArgs = append(containerNamesArgs, container)

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker command: %s", err)
	}

	containerNamesCmd := exec.Command(dockerCmd, containerNamesArgs...)
	containerNamesCmd.Stdout = &output

	if err := containerNamesCmd.Run(); err != nil {
		return fmt.Errorf("Error attempting to get all tags for container: %s", err.Error())
	}

	containerNames = strings.Split(strings.Trim(strings.TrimSpace(output.String()), "'[]"), " ")
	logger.Debug(containerNames)

	for _, name := range containerNames {
		logger.Debugf("Removing container: %s", name)
		rmCmdArgs := []string{"rmi", name}

		rmCmd := exec.Command(dockerCmd, rmCmdArgs...)
		if err := rmCmd.Run(); err != nil {
			return fmt.Errorf("Error removing container %s: %s", name, err.Error())
		}
	}

	return nil
}
