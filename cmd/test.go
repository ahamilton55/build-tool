package cmd

import (
	"build_tool/utils"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const secretsFileKey = "SECRETS_FILE"
const appEnvKey = "APP_ENV"

var secretsFile string
var extraVolumes []string

func init() {
	testCli.Flags().StringVarP(&secretsFile, "secrets-file", "f", "", "Secrets file S3 location")
	testCli.Flags().StringSliceVarP(&extraVolumes, "volume", "v", []string{}, "Extra volumes to add")
	RootCmd.AddCommand(testCli)
}

var testCli = &cobra.Command{
	Use:   "test",
	Short: "Tests a Docker container for a service using phpunit",
	Long:  `Tests a Docker container for a service using phpunit`,
	Run: func(cmd *cobra.Command, args []string) {
		testContainer()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		CmdSetup()
	},
}

func testContainer() {
	var err error

	repoTopLevel, err := utils.GitToplevel()
	if err != nil {
		utils.ErrorAndQuit("Error looking up top level of directory", err, 2)
	}

	testCmdArgs := buildTestCmdArgs(AppEnv, appEnvKey, secretsFile, repoTopLevel, Config.Name, Config.TestScript, extraVolumes)

	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		utils.ErrorAndQuit("Could not find docker command", err, 3)
	}
	testCmd := exec.Command(dockerCmd, testCmdArgs...)
	testCmd.Stdout = os.Stdout

	err = testCmd.Run()
	if err != nil {
		utils.ErrorAndQuit("An error occurred running the test container", err, 3)
	}
}

func getSecretsFile(secretsFile, env, repoTopLevel string) []string {
	cmdArgs := []string{}

	if secretsFile == "" {
		secretsFile = os.Getenv(secretsFileKey)
	}
	if secretsFile != "" {
		// if secretsFile plus the repoToplevel exists, then put those two
		if _, err := os.Stat(fmt.Sprintf("%s/%s", repoTopLevel, secretsFile)); err == nil {
			cmdArgs = append(cmdArgs, "-v", fmt.Sprintf("%s/%s:/tmp/%s", repoTopLevel, secretsFile, env))
			// else if secretsFile exists, then just put it
		} else if _, err := os.Stat(secretsFile); err == nil {
			cmdArgs = append(cmdArgs, "-v", fmt.Sprintf("%s:/tmp/%s", secretsFile, env))
			// else put it as an environment variable
		} else {
			cmdArgs = append(cmdArgs, "-e", fmt.Sprintf("%s=%s", secretsFileKey, secretsFile))
		}
	}

	return cmdArgs
}

func buildTestCmdArgs(env, appEnvKey, secretsFile, repoTopLevel, name, testScript string, extraVolumes []string) []string {
	testCmdArgs := []string{"run", "--rm", "-e", fmt.Sprintf("%s=%s", appEnvKey, env)}

	secretsFileArgs := getSecretsFile(secretsFile, env, repoTopLevel)
	testCmdArgs = append(testCmdArgs, secretsFileArgs...)

	for _, volume := range extraVolumes {
		testCmdArgs = append(testCmdArgs, "-v", volume)
	}

	dockerTag := utils.GetDockerJobTag()

	testCmdArgs = append(testCmdArgs, fmt.Sprintf("%s:%s", name, dockerTag))

	if testScript != "" {
		testCmdArgs = append(testCmdArgs, testScript)
	} else {
		testCmdArgs = append(testCmdArgs, utils.DefaultTestScript)
	}

	return testCmdArgs
}
