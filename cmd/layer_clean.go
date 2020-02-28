// The 'srk layer clean' command. This cleans up any local packages or build
// files for the specified layer.
package cmd

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var layerCleanName string

// layerCleanCmd represents the clean command
var layerCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up local packages and build files",
	Long:  `Clean will remove any local files that were generated for the specified layer (or all layers if no layer-name is provided. Clean does not affect function service providers (use "remove" to remove a layer from a provider)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var cleanGlob string
		if layerCleanName != "" {
			cleanGlob = srkManager.GetRawLayerPath(layerCleanName) + "*"
		} else {
			cleanGlob = filepath.Join(srkManager.Cfg.GetString("buildDir"), "layers", "*")
		}

		if err := srkManager.CleanDirectory(cleanGlob); err != nil {
			return errors.Wrap(err, "Failed to clean layer")
		}

		srkManager.Logger.Info("Successfully cleaned layer")
		return nil
	},
}

func init() {
	layerCmd.AddCommand(layerCleanCmd)

	layerCleanCmd.Flags().StringVarP(&layerCleanName, "layer-name", "n", "", "The layer to clean (defaults to cleaning all packages)")
}
