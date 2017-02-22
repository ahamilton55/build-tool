package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// Looks up a given stack in AWS Cloudformation and returns the tag of the container
// currently running in the stack.
//
// stackName -- Name of the Cloudformation stack to look up
// region -- AWS region to use
// profile -- AWS profile to use
func FindLatestDeployTag(stackName, region, profile string) (string, error) {
	var err error
	var sess *session.Session
	var awsConfig *aws.Config
	var taskId string
	var tag string

	if region != "" {
		awsConfig = &aws.Config{Region: aws.String(region)}
	} else {
		awsConfig = &aws.Config{}
	}

	sess, err = session.NewSessionWithOptions(session.Options{
		Config:            *awsConfig,
		Profile:           "vmnetops",
		SharedConfigState: session.SharedConfigEnable,
	})

	cf := cloudformation.New(sess)
	params := &cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	}

	resp, err := cf.ListStackResources(params)
	if err != nil {
		return "", err
	}

	for _, v := range resp.StackResourceSummaries {
		if aws.StringValue(v.LogicalResourceId) == "taskdefinition" {
			taskId = aws.StringValue(v.PhysicalResourceId)
			break
		}
	}

	ecsClient := ecs.New(sess)

	ecsParams := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskId),
	}

	ecsResp, err := ecsClient.DescribeTaskDefinition(ecsParams)
	if err != nil {
		return "", err
	}

	containerName := strings.Join(strings.Split(stackName, "-")[0:2], "-")
	if len(ecsResp.TaskDefinition.ContainerDefinitions) == 1 {
		tag = strings.Split(aws.StringValue(ecsResp.TaskDefinition.ContainerDefinitions[0].Image), ":")[1]
	} else {
		for _, v := range ecsResp.TaskDefinition.ContainerDefinitions {
			if aws.StringValue(v.Name) == containerName {
				// v.Image is of the form <repo>/<image>:<tag>
				// repo and image cannot have a ":" in them
				// so there are only two splits and we want the second
				tag = strings.Split(aws.StringValue(v.Image), ":")[1]
				break
			}
		}
	}
	return tag, nil
}

func GetAWSSession(region, profile string) (*session.Session, error) {
	var awsConfig *aws.Config

	if region != "" {
		awsConfig = &aws.Config{Region: aws.String(region)}
	} else {
		awsConfig = &aws.Config{}
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            *awsConfig,
		Profile:           "vmnetops",
		SharedConfigState: session.SharedConfigEnable,
	})

	return sess, err
}

func getRegistryId(ecr string) string {
	ecrParts := strings.Split(ecr, ".")

	return ecrParts[0]
}

// Looks up a given stack in an AWS ECR and returns the tag of the container
// that was most recently built.
//
// ecr -- Name of the AWS ECR to use
// region -- AWS region to use
// profile -- AWS profile to use
func FindLatestBuildTag(ecrRepo, name, env string, sess *session.Session) (string, error) {
	var (
		nextToken string
		tagRegExp string
		matches   []string
	)

	registryId := getRegistryId(ecrRepo)

	client := ecr.New(sess)

	if env == "prod" {
		tagRegExp = fmt.Sprintf(TagPassRegex, "stage")
	} else {
		tagRegExp = TagDefaultRegex
	}

	for {
		params := &ecr.ListImagesInput{
			RepositoryName: aws.String(name),
			RegistryId:     aws.String(registryId),
		}

		if nextToken != "" {
			params.NextToken = aws.String(nextToken)
		}

		resp, err := client.ListImages(params)
		if err != nil {
			return "", err
		}

		for _, v := range resp.ImageIds {
			if aws.StringValue(v.ImageTag) == "" {
				continue
			}
			tag := aws.StringValue(v.ImageTag)
			match, err := regexp.MatchString(tagRegExp, tag)
			if err != nil {
				return "", err
			}

			if match {
				matches = append(matches, tag)
			}
		}

		if resp.NextToken != nil {
			nextToken = *resp.NextToken
		} else {
			break
		}
	}

	// Sort the matches and then return the last one which will be the latest
	sort.Strings(matches)

	if len(matches) <= 0 {
		return "", fmt.Errorf("No matching containers found")
	} else {
		return matches[len(matches)-1], nil
	}
}

// Sets ECR Login credentials on the host for pushing and pulling docker
// containers
//
// region -- AWS region to use
// profile -- AWS profile to use
// ecr -- Name of the ECR to use
func EcrLogin(region, profile, ecr string) error {
	var err error
	loginCmdArgs := []string{"ecr", "get-login"}

	if region != "" {
		loginCmdArgs = append(loginCmdArgs, "--region", region)
	}

	if profile != "" {
		loginCmdArgs = append(loginCmdArgs, "--profile", profile)
	}

	registryId := strings.Split(ecr, ".")[0]
	loginCmdArgs = append(loginCmdArgs, "--registry-ids", registryId)

	awsCmd, err := exec.LookPath("aws")
	if err != nil {
		return fmt.Errorf("Could not find aws command: %s")
	}

	loginCmd := exec.Command(awsCmd, loginCmdArgs...)

	var out bytes.Buffer
	loginCmd.Stdout = &out

	err = loginCmd.Run()
	if err != nil {
		return fmt.Errorf("Error getting login credentials for AWS ECR: %s", err)
	}

	outputSlice := strings.Split(strings.TrimSpace(out.String()), " ")
	dockerLoginArgs := outputSlice[1:]
	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Could not find docker command: %s", err)
	}

	dockerLoginCmd := exec.Command(dockerCmd, dockerLoginArgs...)
	dockerLoginCmd.Stdout = os.Stdout
	dockerLoginCmd.Stderr = os.Stderr

	err = dockerLoginCmd.Run()
	if err != nil {
		return fmt.Errorf("Error setting up login credentials with Docker: %s", err)
	}

	return nil
}
