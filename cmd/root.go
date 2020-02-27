// Root of command-line argument parsing.
// This file was based off the standard cobra template, see
// https://github.com/spf13/cobra
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/serverlessresearch/srk/pkg/srkmgr"
	"github.com/spf13/cobra"
)

var cfgFile string

var srkManager *srkmgr.SrkManager

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "srk",
	Short: "The Berkeley Serverless Research Kit",
	Long:  `A collection of tools for experimenting with serverless systems.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		mgrArgs := map[string]interface{}{}
		if cfgFile != "" {
			mgrArgs["config-file"] = cfgFile
		}

		var err error
		srkManager, err = srkmgr.NewManager(mgrArgs)
		if err != nil {
			fmt.Println("Failed to initialize srk manager: %v\n", err)
			os.Exit(1)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		srkManager.Destroy()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by srk.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if srkManager == nil || srkManager.Logger == nil {
			fmt.Printf("%v\n", err)
		} else {
			srkManager.Logger.Error(err)
		}
		os.Exit(1)
	}
}

func parseKeyValue(s string) map[string]string {

	if s == "" {
		return nil
	}

	result := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) == 2 {
			result[keyValue[0]] = keyValue[1]
		}
	}

	return result
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is configs/srk.yaml)")
}
