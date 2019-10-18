// Root of command-line argument parsing.
// This file was based off the standard cobra template, see
// https://github.com/spf13/cobra
package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	awslambda "github.com/serverlessresearch/srk/pkg/aws-lambda"
	"github.com/serverlessresearch/srk/pkg/openlambda"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	// The configuration management library
	"github.com/spf13/viper"
)

var cfgFile string

var srkConfig struct {
	provider *srk.Provider
	logger   *logrus.Logger
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "srk",
	Short: "The Berkeley Serverless Research Kit",
	Long:  `A collection of tools for experimenting with serverless systems.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// This must be done in a PreRun because it may load from viper in the
		// future.
		srkConfig.logger = logrus.New()

		// Ideally, this would be logged earlier, but we had to wait until
		// logging was enabled.
		srkConfig.logger.Info("Using config file: ", viper.ConfigFileUsed())

		faas, err := getFunctionService()
		if err != nil {
			// return errors.Wrap(err, "Failed to initialize srk provider")
			fmt.Printf("%v\n", err)
			panic("Failed to initialize srk provider\n")
		}
		srkConfig.provider = &srk.Provider{
			Faas: faas,
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		srkConfig.provider.Faas.Destroy()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by srk.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		srkConfig.logger.Error(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is configs/srk.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Setup defaults and globals here. These can be overwritten in the config,
	// but aren't included by default.

	// Dumping ground for all generated output. Users should be able to "rm -rf
	// build" and get a clean system.
	viper.SetDefault("buildDir", "./build")

	// Collects all srk-provided libraries.
	viper.SetDefault("includeDir", "./includes")

	// Order of precedence: ENV, srk.yaml, "us-west-2"
	viper.SetDefault("service.faas.awsLambda.region", "us-west-2")
	viper.BindEnv("service.faas.awsLambda.region", "AWS_DEFAULT_REGION")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// default search path for config is ./configs/srk.* (* can be json, yaml, etc)
		viper.AddConfigPath("./configs")
		viper.SetConfigName("srk")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		panic(err)
	}
}

func getFunctionService() (srk.FunctionService, error) {
	// Setup the default function service
	providerName := viper.GetString("default-provider")
	if providerName == "" {
		return nil, errors.New("No default provider in configuration")
	}

	serviceName := viper.GetString("providers." + providerName + ".faas")
	if serviceName == "" {
		return nil, errors.New("Provider \"" + providerName + "\" does not provide a FaaS service")
	}

	var service srk.FunctionService
	switch serviceName {
	case "openLambda":
		// service = openlambda.NewConfig(viper.Sub("service.faas.openLambda"), )
		service = openlambda.NewConfig(
			viper.GetString("service.faas.openLambda.olcmd"),
			viper.GetString("service.faas.openLambda.oldir"))
	case "awsLambda":
		service = awslambda.NewConfig(
			viper.GetString("service.faas.awsLambda.role"),
			viper.GetString("service.faas.awsLambda.vpc-config"),
			viper.GetString("service.faas.awsLambda.region"))
	default:
		return nil, errors.New("Unrecognized FaaS service: " + serviceName)
	}

	return service, nil
}
