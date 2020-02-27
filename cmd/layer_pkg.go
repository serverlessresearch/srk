// Package creates a layer archive that can be installed to a service.
package cmd

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var layerPackageCmdConfig struct {
	source string
	files  string
	name   string
}

var layerPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package the files needed to install a layer, but does not actually install it.",
	Long: `Each FaaS service provider has their own format and requirements on
a layer package. Typically, these take the form of an archive (e.g. .tgz or
.zip). The command will tell you where the package was saved so that you
can manually inspect or modify it.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if layerPackageCmdConfig.source == "" {
			return errors.New("source is required")
		}

		if layerPackageCmdConfig.name == "" {
			layerPackageCmdConfig.name = strings.TrimSuffix(path.Base(layerPackageCmdConfig.source), path.Ext(layerPackageCmdConfig.source))
		}

		files := parseList(layerPackageCmdConfig.files)

		rawDir, err := srkManager.CreateRawLayer(layerPackageCmdConfig.source, layerPackageCmdConfig.name, files)
		if err != nil {
			return errors.Wrap(err, "Packaging layer failed")
		}
		srkManager.Logger.Info("Created raw layer: " + rawDir)

		pkgPath, err := srkManager.Provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaging failed")
		}
		srkManager.Logger.Info("Package created at: " + pkgPath)
		return nil
	},
}

func init() {
	layerCmd.AddCommand(layerPackageCmd)

	// Define the command line arguments for this subcommand
	layerPackageCmd.Flags().StringVarP(&layerPackageCmdConfig.source, "source", "s", "", "source directory or file")
	layerPackageCmd.Flags().StringVarP(&layerPackageCmdConfig.files, "files", "f", "", "List of files to include")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	layerPackageCmd.Flags().StringVarP(&layerPackageCmdConfig.name, "layer-name", "n", "", "optional name for this layer")
}
