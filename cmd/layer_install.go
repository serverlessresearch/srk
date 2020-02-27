//Implements the 'srk layer install' subcommand. Install takes a
//pre-packaged layer and actually installs it to the function service.
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var layerInstallCmdConfig struct {
	name     string
	runtimes string
}

// installCmd represents the install command
var layerInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a pre-packaged layer to the configured FaaS service",
	Long:  `Install a layer to the FaaS service. It is assumed that you have already packaged this layer (using the 'package' command).`,
	RunE: func(cmd *cobra.Command, args []string) error {

		rawDir := srkManager.GetRawLayerPath(layerInstallCmdConfig.name)
		compatibleRuntimes := parseList(layerInstallCmdConfig.runtimes)

		layerId, err := srkManager.Provider.Faas.InstallLayer(rawDir, compatibleRuntimes)
		if err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkManager.Logger.Infof("Successfully installed layer: %s", layerId)
		return nil
	},
}

func init() {
	layerCmd.AddCommand(layerInstallCmd)

	layerInstallCmd.Flags().StringVarP(&layerInstallCmdConfig.name, "layer-name", "n", "", "The layer to install")
	layerInstallCmd.Flags().StringVarP(&layerInstallCmdConfig.runtimes, "compatible-runtimes", "r", "", "List of compatible runtimes")
}
