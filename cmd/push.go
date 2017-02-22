package cmd

import (
	"build_tool/utils"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	tag           string
	containerName string
)

func init() {
	pushCli.Flags().StringVarP(&tag, "tag", "t", "", "tag to push. Default: current build tag")
	pushCli.Flags().StringVarP(&containerName, "container", "c", "", "full name of the container to push to ECR")
	RootCmd.AddCommand(pushCli)
}

var pushCli = &cobra.Command{
	Use:   "push",
	Short: "Pushes a Docker container to ECR",
	Long:  `Pushes a Docker container to ECR`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("Starting container push")
		pushContainer()
		logger.Debug("Completed container push")
	},
}

func pushContainer() {
	var err error

	if err = utils.EcrLogin(Region, Profile, Config.EcrRepo); err != nil {
		logger.Debug("Login to ECR")
		utils.ErrorAndQuit("", err, 2)
	}

	if tag == "" {
		logger.Debug("Looking up docker tag")
		tag = utils.GetDockerJobTag()
	}

	if containerName == "" {
		logger.Debug("Building remote container name")
		containerName = fmt.Sprintf("%s/%s:%s", Config.EcrRepo, Config.Name, tag)
	}

	logger.Debugf("Pushing %s\n", containerName)
	if err = utils.Push(containerName); err != nil {
		utils.ErrorAndQuit("", err, 4)
	}
}
