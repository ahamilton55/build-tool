package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"build_tool/utils"

	"github.com/spf13/cobra"
)

var (
	DockerOutput    bool
	dockerBuildArgs string
)

func init() {
	buildCli.PersistentFlags().BoolVar(&DockerOutput, "docker-output", false, "Print docker build output to STDERR")
	buildCli.PersistentFlags().StringVar(&dockerBuildArgs, "docker-args", "", "Extra arguments to be passed to a docker build")
	RootCmd.AddCommand(buildCli)
}

var buildCli = &cobra.Command{
	Use:   "build",
	Short: "Builds a Docker container for a service",
	Long:  `Builds a Docker container for a service`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("Build starting")

		build()

		logger.Debug("Build completed")
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		CmdSetup()
	},
}

func build() {
	var (
		err           error
		containerName string
		dockerfile    string
		labels        []string
	)

	repoToplevel, err := utils.GitToplevel()
	if err != nil {
		utils.ErrorAndQuit("Error looking up top level of directory", err, 2)
	}

	logger.Debug("Looking up dockerfile")
	dockerfile, err = findDockerfile(repoToplevel)
	if err != nil {
		utils.ErrorAndQuit("", err, 2)
	}

	logger.Debug("Looking up job tag for container")
	dockerTag := utils.GetDockerJobTag()

	logger.Debug("Setup name for container")
	containerName = fmt.Sprintf("%s:%s", Config.Name, dockerTag)

	logger.Debug("Looking up SHA for the HEAD of the repo")
	headSHA, err := utils.GitSHA("HEAD")
	if err != nil {
		utils.ErrorAndQuit("Error looking up the git SHA", err, 2)
	}

	logger.Debug("Setup labels for the container")
	labels = append(labels, fmt.Sprintf("%s=%s", utils.CommitLabel, headSHA))
	labels = append(labels, fmt.Sprintf("%s=%s", utils.BuildDateLabel, time.Now().Format(utils.BuildDateFormat)))
	if len(Config.Labels) > 0 {
		for _, label := range Config.Labels {
			labels = append(labels, label)
		}
	}

	logger.Debug("Building container")
	if err := buildContainer(containerName, dockerfile, dockerBuildArgs, labels); err != nil {
		utils.ErrorAndQuit("Unable to build service container", err, 4)
	}

}

func buildContainer(containerName, dockerfile, dockerBuildArgs string, labels []string) error {
	logger.Debug("Setup docker build command arguments")
	buildCmdArgs := []string{"build", "-t", containerName}
	for _, label := range labels {
		buildCmdArgs = append(buildCmdArgs, "--label", label)
	}

	if len(dockerBuildArgs) > 0 {
		for _, arg := range strings.Split(dockerBuildArgs, " ") {
			buildCmdArgs = append(buildCmdArgs, arg)
		}
	}

	buildCmdArgs = append(buildCmdArgs, "-f", dockerfile, ".")

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker executable")
	}
	logger.Debug("Creating docker build command")
	buildCmd := exec.Command(dockerCmd, buildCmdArgs...)
	if DebugOutput || DockerOutput {
		logger.Debug("Setting docker build output and errors to os.Stderr")
		buildCmd.Stdout = os.Stderr
		buildCmd.Stderr = os.Stderr
	}

	logger.Debug("Running docker build command")
	err = buildCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func findDockerfile(repoToplvl string) (string, error) {
	var dockerfile string

	if Config.Dockerfile != "" {
		dockerfile = strings.Join([]string{repoToplvl, Config.Dockerfile}, "/")
	} else {
		dockerfile = strings.Join([]string{repoToplvl, utils.DefaultDockerfile}, "/")
	}

	if _, err := os.Stat(dockerfile); err != nil {
		return "", fmt.Errorf("Could not find Dockerfile at %s", dockerfile)
	}

	return dockerfile, nil
}
