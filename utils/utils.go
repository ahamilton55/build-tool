package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	BuildDateFormat        = "0601021504" // Not a string! It's a format string yymmddHHMM
	defaultEnvSettingsFile = "deploy/env_setup"
	defaultTag             = "latest"

	BuildDateLabel = "com.katch.build_date"
	CommitLabel    = "com.katch.commit"

	TagDeployFmt = "%s-deploy"
	TagPassFmt   = "%s-pass"
	TagFailFmt   = "%s-fail"

	TagDeployRegex  = "^%s-deploy-[0-9]*$"
	TagPassRegex    = "^%s-pass-[0-9]*$"
	TagFailRegex    = "^%s-fail-[0-9]*$"
	TagDefaultRegex = "^[0-9]*$"
)

// Print out an error and then quit with the given exit code.
func ErrorAndQuit(msg string, err error, exitCode int) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
	os.Exit(exitCode)
}

// Check to see if the program is currently running at the top-level of
// a git repository.
func CheckAtToplevel() bool {
	repoToplevel, err := GitToplevel()
	if err != nil {
		return false
	}

	pwd, err := os.Getwd()
	if err != nil {
		return false
	}

	if pwd != repoToplevel {
		return false
	}

	return true
}

// Creates a set of environment settings based off a default file location.
func GetEnvSettings() (map[string]string, error) {
	envSettings := make(map[string]string)

	repoToplevel, err := GitToplevel()
	if err != nil {
		return envSettings, fmt.Errorf("Error looking up top level of directory: %s", err)
	}
	envFile := strings.Join([]string{repoToplevel, defaultEnvSettingsFile}, "/")
	if _, err := os.Stat(envFile); err != nil {
		return envSettings, fmt.Errorf("Error looking up env file: %s", err)
	}

	data, err := ioutil.ReadFile(envFile)
	if err != nil {
		return envSettings, fmt.Errorf("Error reading env file: %s", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		lineSplit := strings.Split(line, "=")
		if len(lineSplit) >= 2 {

			value := strings.Join(lineSplit[1:], "=")
			value = strings.Split(value, "#")[0] // remove comments at the end of a line
			value = strings.TrimPrefix(value, `"`)
			value = strings.TrimSuffix(value, `"`)
			envSettings[lineSplit[0]] = value
		}
	}

	return envSettings, nil
}

// Generates a Docker job tag when running in a CI environment.
func GetDockerJobTag() string {
	var dockerTag string
	if os.Getenv("JOB_NAME") != "" && os.Getenv("BUILD_NUMBER") != "" {
		dockerTag = strings.Join([]string{os.Getenv("JOB_NAME"), os.Getenv("BUILD_NUMBER")}, "-")
		dockerTag = strings.Replace(dockerTag, "/", "_", -1)
		dockerTag = strings.Replace(dockerTag, "%2F", "_", -1)
	} else {
		dockerTag = defaultTag
	}
	return dockerTag
}

// Creates a default set of tags with the given input
func CreateTag(env, date string, successful, failure, deploy bool) string {
	if date == "" {
		date = time.Now().Format(BuildDateFormat)
	}

	if deploy {
		return fmt.Sprintf("%s-%s", fmt.Sprintf(TagDeployFmt, env), date)
	} else if successful {
		return fmt.Sprintf("%s-%s", fmt.Sprintf(TagPassFmt, env), date)
	} else if failure {
		return fmt.Sprintf("%s-%s", fmt.Sprintf(TagFailFmt, env), date)
	}

	return date
}

// Returns the name of a Task based on the environment and stack.
func GetTaskStackName(env, stack string) string {
	return fmt.Sprintf("%s-%s", env, stack)
}

// Returns the commit label for the current repository
func GetCommitLabel() (string, error) {
	headSHA, err := GitSHA("HEAD")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s=%s", CommitLabel, headSHA), nil
}
