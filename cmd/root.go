// Root of command-line argument parsing.
// This file was based off the standard cobra template, see
// https://github.com/spf13/cobra
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// The configuration management library
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "srk",
	Short: "The Berkeley Serverless Research Kit",
	Long:  `A collection of tools for experimenting with serverless systems.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by srk.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for the config at ./configs/user/srk.* first, if not found,
		// default to ./configs/srk.* (* can be json, yaml, etc)
		viper.AddConfigPath("./configs/user")
		viper.AddConfigPath("./configs")
		viper.SetConfigName("srk")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Printf("Failed to load config: %v\n", err)
		panic(err)
	}
}
