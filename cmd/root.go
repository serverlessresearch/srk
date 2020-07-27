// Root of command-line argument parsing.
// This file was based off the standard cobra template, see
// https://github.com/spf13/cobra
package cmd

import (
	"os"

	"github.com/serverlessresearch/srk/pkg/srkmgr"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var srkHome string

var srkManager *srkmgr.SrkManager

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "srk",
	Short: "The Berkeley Serverless Research Kit",
	Long:  `A collection of tools for experimenting with serverless systems.`,

	// TODO - have this commented out right now because we're bringing in configuration not needed for LambdaLike

	// PersistentPreRun: func(cmd *cobra.Command, args []string) {
	// 	mgrArgs := map[string]interface{}{}
	// 	if srkHome != "" {
	// 		mgrArgs["srk-home"] = srkHome
	// 	}

	// 	var err error
	// 	srkManager, err = srkmgr.NewManager(mgrArgs)
	// 	if err != nil {
	// 		log.Fatalf("Failed to initialize srk manager: %v\n", err)
	// 	}
	// },
	// PersistentPostRun: func(cmd *cobra.Command, args []string) {
	// 	srkManager.Destroy()
	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by srk.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if srkManager == nil || srkManager.Logger == nil {
			log.Error(err)
		} else {
			srkManager.Logger.Error(err)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&srkHome, "srk-home", "", "SRK install directory. Defaults to the SRKHOME environment variable (or ./runtime if not provided).")
}
