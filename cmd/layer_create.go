// Handles the "srk layer create" command

package cmd

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var layerCreateCmdConfig struct {
	source   string
	name     string
	files    string
	runtimes string
}

var layerCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new layer and upload it to the FaaS provider",
	Long: `This will package up your layer and upload it to the configured
FaaS provider. Create is equivalent to calling "srk
layer package" and "srk layer install".`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if layerCreateCmdConfig.source == "" {
			return errors.New("source is required")
		}

		if layerCreateCmdConfig.name == "" {
			layerCreateCmdConfig.name = strings.TrimSuffix(path.Base(layerCreateCmdConfig.source), path.Ext(layerCreateCmdConfig.source))
		}

		files := parseList(layerCreateCmdConfig.files)

		rawDir, err := srkManager.CreateRawLayer(layerCreateCmdConfig.source, layerCreateCmdConfig.name, files)
		if err != nil {
			return errors.Wrap(err, "Creating raw layer failed")
		}
		srkManager.Logger.Info("Created raw layer: " + rawDir)

		pkgPath, err := srkManager.Provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaging failed")
		}
		srkManager.Logger.Info("Package created at: " + pkgPath)

		compatibleRuntimes := parseList(layerCreateCmdConfig.runtimes)
		layerId, err := srkManager.Provider.Faas.InstallLayer(rawDir, compatibleRuntimes)
		if err != nil {
			return errors.Wrap(err, "Installation failed")
		}

		srkManager.Logger.Infof("Successfully installed layer: %s", layerId)
		return nil
	},
}

func init() {
	layerCmd.AddCommand(layerCreateCmd)

	// Define the command line arguments for this subcommand
	layerCreateCmd.Flags().StringVarP(&layerCreateCmdConfig.source, "source", "s", "", "source directory or file")
	layerCreateCmd.Flags().StringVarP(&layerCreateCmdConfig.files, "files", "f", "", "List of files to include")
	layerCreateCmd.Flags().StringVarP(&layerCreateCmdConfig.runtimes, "compatible-runtimes", "r", "", "List of compatible runtimes")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	layerCreateCmd.Flags().StringVarP(&layerCreateCmdConfig.name, "layer-name", "n", "", "optional name for this layer")
}
