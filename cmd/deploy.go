package cmd

import (
	"build_tool/utils"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/spf13/cobra"
)

var (
	container string
	newStack  bool
)

const defaultSleepTime = 5

func init() {
	deployCli.Flags().StringVarP(&container, "container", "c", "", "container to use for deploy")
	deployCli.Flags().BoolVarP(&newStack, "new-stack", "n", false, "create a new stack if one does not exist")
	RootCmd.AddCommand(deployCli)
}

var deployCli = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an ECR service with Cloudformation",
	Long:  `Deploy an ECR service with Cloudformation`,
	Run: func(cmd *cobra.Command, args []string) {
		deploy()
	},
}

func deploy() {
	var parameters []*cloudformation.Parameter

	sess, err := utils.GetAWSSession(Region, Profile)
	if err != nil {
		utils.ErrorAndQuit("Error getting AWS Session", err, 3)
	}

	cf := cloudformation.New(sess)

	stackName := fmt.Sprintf("%s-%s", AppEnv, Config.Stack)

	_, err = cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		newStack = true
	} else if err != nil {
		utils.ErrorAndQuit("Error checking the stack's status", err, 5)
	}

	container, err = findContainer(container, Config.EcrRepo, Config.Name, AppEnv, sess)
	if err != nil {
		utils.ErrorAndQuit("No container provided and could not find container", err, 5)
	}

	if Config.CFTemplate == "" {
		utils.ErrorAndQuit("No CF template set", nil, 3)
	}

	parameters = setupParameters(container, Config)

	err = launchStack(newStack, stackName, Config.CFTemplate, parameters, cf)
	if err != nil && strings.Contains(err.Error(), "No updates are to be performed") {
		logger.Info("Nothing to update")
		os.Exit(255)
	} else if err != nil {
		utils.ErrorAndQuit("Unable to setup stack", err, 6)
	} else {
		if err = watchStack(stackName, defaultSleepTime, cf); err != nil {
			utils.ErrorAndQuit("Stack creation/update was not successful", err, 7)
		}
	}
}

func setupParameters(name string, config utils.Config) []*cloudformation.Parameter {
	var parameters []*cloudformation.Parameter

	for _, val := range config.CFParameters[AppEnv] {
		parameter := strings.Split(val, "=")
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(parameter[0]),
			ParameterValue: aws.String(parameter[1]),
		})
	}

	parameters = append(parameters, &cloudformation.Parameter{
		ParameterKey:   aws.String("ImageID"),
		ParameterValue: aws.String(container),
	})

	parameters = append(parameters, &cloudformation.Parameter{
		ParameterKey:   aws.String("TaskName"),
		ParameterValue: aws.String(fmt.Sprintf("%s-%s", AppEnv, config.Stack)),
	})

	return parameters
}

func findContainer(container, ecrRepo, name, env string, sess *session.Session) (string, error) {
	if container == "" {
		latestBuildTag, err := utils.FindLatestBuildTag(ecrRepo, name, env, sess)
		if err != nil {
			return "", fmt.Errorf("Error looking up latest build tag")
		}
		return fmt.Sprintf("%s/%s:%s", ecrRepo, name, latestBuildTag), nil
	}

	return container, nil
}

func launchStack(newStack bool, stackName, cfTemplate string, parameters []*cloudformation.Parameter, cf *cloudformation.CloudFormation) error {
	if newStack {
		stackInput := cloudformation.CreateStackInput{}

		stackInput.Parameters = parameters
		stackInput.StackName = aws.String(stackName)
		if strings.Contains(Config.CFTemplate, "s3://") {
			stackInput.TemplateURL = aws.String(cfTemplate)
		} else {
			contents, err := ioutil.ReadFile(cfTemplate)
			if err != nil {
				return err
			}
			stackInput.TemplateBody = aws.String(string(contents))
		}

		_, err := cf.CreateStack(&stackInput)
		if err != nil {
			return err
		}
	} else {
		stackInput := cloudformation.UpdateStackInput{}

		stackInput.Parameters = parameters
		stackInput.StackName = aws.String(stackName)
		if strings.Contains(Config.CFTemplate, "s3://") {
			stackInput.TemplateURL = aws.String(cfTemplate)
		} else {
			contents, err := ioutil.ReadFile(cfTemplate)
			if err != nil {
				return err
			}
			stackInput.TemplateBody = aws.String(string(contents))
		}
		_, err := cf.UpdateStack(&stackInput)
		if err != nil {
			return err
		}
	}

	return nil
}

func watchStack(stackName string, sleepTime int, cf *cloudformation.CloudFormation) error {
Loop:
	for {
		resp, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			return err
		}

		switch aws.StringValue(resp.Stacks[0].StackStatus) {
		case cloudformation.StackStatusCreateFailed:
			return fmt.Errorf("Failed to create the stack")
		case cloudformation.StackStatusRollbackFailed:
			return fmt.Errorf("Failed to rollback the stack")
		case cloudformation.StackStatusUpdateRollbackFailed:
			return fmt.Errorf("Failed to rollback the stack update")
		case cloudformation.StackStatusUpdateRollbackComplete:
			return fmt.Errorf("Stack update failed and rolledback")
		case cloudformation.StackStatusCreateComplete:
			break Loop
		case cloudformation.StackStatusUpdateComplete:
			break Loop
		}

		time.Sleep(time.Duration(sleepTime) * time.Second)
	}

	return nil
}
