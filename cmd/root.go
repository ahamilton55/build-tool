package cmd

import (
	"build_tool/utils"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	Region      string
	Profile     string
	AppEnv      string
	ConfigFile  string
	DebugOutput bool

	logger *log.Entry

	Config utils.Config
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&Region, "region", "r", "", "AWS region to use")
	RootCmd.PersistentFlags().StringVarP(&Profile, "profile", "p", "", "AWS profile to use")
	RootCmd.PersistentFlags().StringVarP(&AppEnv, "env", "e", "", "Application environment")
	RootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "", utils.DefaultConfigFile, "Location of config file to use")
	RootCmd.PersistentFlags().BoolVarP(&DebugOutput, "debug", "", false, "Print debugging info to stderr")
}

var RootCmd = &cobra.Command{
	Use:   "build_tool",
	Short: "build_tool is a glorified shell script ported to Go",
	Run: func(cmd *cobra.Command, args []string) {
		// Do stuff here
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if AppEnv == "" {
			utils.ErrorAndQuit("You must provide an environment", nil, 1)
		}

		CmdSetup()
	},
}

func CmdSetup() {
	var err error

	log.SetOutput(os.Stderr)
	if DebugOutput {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.ErrorLevel)
	}

	Config, err = utils.ReadConfig(ConfigFile)
	if err != nil {
		utils.ErrorAndQuit("Error looking up env settings", err, 2)
	}

	logger = log.WithFields(log.Fields{
		"app_env":     AppEnv,
		"aws_profile": Profile,
		"aws_region":  Region,
	})

	if !utils.CheckAtToplevel() {
		utils.ErrorAndQuit("Please run this command from the root of the repo", nil, 2)
	}

	if Config.Name == "" {
		utils.ErrorAndQuit("Name not supplied in the config", nil, 2)
	}

	if Config.EcrRepo == "" {
		utils.ErrorAndQuit("ECR Repo not found in the config", nil, 2)
	}

}
