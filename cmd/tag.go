package cmd

import (
	"build_tool/utils"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	oldTag string
	newTag string
	date   string

	successful       bool
	failure          bool
	deployTag        bool
	findLatestDeploy bool
	localContainer   bool
	outputNewName    bool
)

func init() {
	tagCli.Flags().StringVarP(&oldTag, "old-tag", "o", "", "old tag")
	tagCli.Flags().StringVarP(&newTag, "new-tag", "n", "", "new tag")
	tagCli.Flags().BoolVarP(&successful, "successful", "s", false, "tag the container as successful")
	tagCli.Flags().BoolVarP(&failure, "failure", "f", false, "tag the container as successful")
	tagCli.Flags().BoolVarP(&deployTag, "deploy", "d", false, "tag the container as deployed")
	tagCli.Flags().StringVarP(&date, "time", "t", "", "date and time for a the tag")
	tagCli.Flags().BoolVarP(&findLatestDeploy, "find-latest-deploy", "", false, "look up the latest deployed container for a given env")
	tagCli.Flags().BoolVarP(&localContainer, "local", "", false, "Use a local container name")
	tagCli.Flags().BoolVarP(&outputNewName, "output-new-name", "", false, "Print the new container name to STDOUT")
	RootCmd.AddCommand(tagCli)
}

var tagCli = &cobra.Command{
	Use:   "tag",
	Short: "Tags a Docker container",
	Long:  `Tags a Docker container`,
	Run: func(cmd *cobra.Command, args []string) {
		tagContainer()
	},
}

func tagContainer() {
	var err error
	var oldContainer string
	var newContainer string

	if err = utils.EcrLogin(Region, Profile, Config.EcrRepo); err != nil {
		logger.Debug("Login to ECR")
		utils.ErrorAndQuit("", err, 2)
	}

	logger.Debug("Getting original docker container name")
	oldContainer, err = oldContainerName(AppEnv, Region, Profile, Config.Stack, Config.Name, Config.EcrRepo)
	if err != nil {
		utils.ErrorAndQuit("Could not lookup original container name", err, 3)
	}
	logger.Debugf("Old container: %s", oldContainer)

	logger.Debug("Creating new container name")
	newContainer, err = newContainerName(AppEnv, date, Config.EcrRepo, Config.Name, successful, failure, deployTag)
	if err != nil {
		utils.ErrorAndQuit("Could not create the new container's name", err, 3)
	}
	logger.Debugf("New container: %s", newContainer)

	logger.Debug("Tagging container with the new name")
	if err := utils.TagContainer(oldContainer, newContainer, Region, Profile); err != nil {
		utils.ErrorAndQuit("Failed tagging container", err, 4)
	}

	if outputNewName {
		fmt.Println(newContainer)
	}
}

func newContainerName(env, date, ecrRepo, name string, successful, failure, deployTag bool) (string, error) {
	if newTag == "" {
		newTag = utils.CreateTag(env, date, successful, failure, deployTag)
	}
	logger.Debug("Building out the new container name")
	return fmt.Sprintf("%s/%s:%s", Config.EcrRepo, Config.Name, newTag), nil
}

func oldContainerName(env, region, profile, stack, name, ecrRepo string) (string, error) {
	var (
		err          error
		oldContainer string
	)

	if oldTag == "" {
		if findLatestDeploy {
			logger.Debug("Looking up the latest deploy for container tag")
			logger.Debug("Looking up task in cloudformation stack")
			stackName := utils.GetTaskStackName(env, stack)

			logger.Debug("Looking for latest container tag in the discovered stack")
			oldTag, err = utils.FindLatestDeployTag(stackName, region, profile)
			if err != nil {
				return "", err
			}
		} else {
			logger.Debug("Creating container tag from the local environment")
			oldTag = utils.GetDockerJobTag()
		}
	}

	logger.Debug("Building out the old container name")
	if localContainer {
		oldContainer = fmt.Sprintf("%s:%s", name, oldTag)
	} else {
		oldContainer = fmt.Sprintf("%s/%s:%s", ecrRepo, name, oldTag)
	}

	return oldContainer, nil
}
