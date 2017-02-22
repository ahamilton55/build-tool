package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	DefaultConfigFile = ".deploy/config.toml"
	DefaultDockerfile = ".deploy/Dockerfile"
	DefaultTestScript = ".deploy/tests.sh"
)

// Info from config file
type Config struct {
	Name         string              // Name of the service that will be created. Used for the name of the container
	EcrRepo      string              // AWS Elastic Container Service Repository to use
	Stack        string              // Name of the stack without the environment. Environment will be added later
	CFTemplate   string              // Cloudformation template to use. S3 based should start with s3://
	CFParameters map[string][]string // A set of key:value pairs for use with Cloudformation
	TestScript   string              // Script used to execute tests. This should be relative to the Dockerfile WORKDIR
	Dockerfile   string              // Should be relative to the repo root
	Labels       []string            // A list of static labels to add to the docker container
}

func getConfigfile(configFile string) string {
	if configFile == "" {
		configFile = DefaultConfigFile
	}
	repoTopLevel, err := GitToplevel()
	if err != nil {
		return DefaultConfigFile
	}

	return strings.Join([]string{repoTopLevel, configFile}, "/")
}

// Reads info from config file
func ReadConfig(configFile string) (Config, error) {
	var config Config
	configFile = getConfigfile(configFile)
	_, err := os.Stat(configFile)
	if err != nil {
		return config, fmt.Errorf("Config file is missing: %s", configFile)
	}

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return config, err
	}
	//log.Print(config.Index)
	return config, nil
}
