// The 'srk function remove' command. This uninstalls a function from the service.
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var layerRemoveName string

// removeCmd represents the remove command
var layerRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Uninstall (remove) a layer from the service provider.",
	Long:  `Remove will delete a layer from the configured provider so that is no longer visible or using resources. If you have installed the layer to multiple services, you will need to call "remove" on each service separately. Remove is the inverse of "install", it does not affect packages.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if err := srkManager.Provider.Faas.RemoveLayer(layerRemoveName); err != nil {
			return errors.Wrap(err, "Layer removal failed")
		}

		srkManager.Logger.Info("Successfully removed layer")
		return nil
	},
}

func init() {
	layerCmd.AddCommand(layerRemoveCmd)

	layerRemoveCmd.Flags().StringVarP(&layerRemoveName, "layer-name", "n", "", "The layer to remove")
	layerRemoveCmd.MarkFlagRequired("layer-name")
}
