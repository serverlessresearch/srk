// The 'srk function clean' command. This cleans up any local packages or build
// files for the specified function.
package cmd

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cleanName string

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up local packages and build files",
	Long:  `Clean will remove any local files that were generated for the specified function (or all functions if no function-name is provided. Clean does not affect function service providers (use "remove" to remove a function from a provider)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var cleanGlob string
		if cleanName != "" {
			cleanGlob = srkManager.GetRawPath(cleanName) + "*"
		} else {
			cleanGlob = filepath.Join(srkManager.Cfg.GetString("buildDir"), "functions", "*")
		}

		if err := srkManager.CleanDirectory(cleanGlob); err != nil {
			return errors.Wrap(err, "Failed to clean function")
		}

		srkManager.Logger.Info("Successfully cleaned function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().StringVarP(&cleanName, "function-name", "n", "", "The function to clean (defaults to cleaning all packages)")
}
