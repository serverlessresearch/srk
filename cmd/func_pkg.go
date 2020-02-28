// Package creates a function archive that can be installed to a service.
package cmd

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var funcPackageCmdConfig struct {
	source  string
	include string
	files   string
	name    string
}

// funcPackageCmd represents the package command
var funcPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package creates all the files needed to install a function, but does not actually install it.",
	Long: `Each FaaS service provider has their own format and requirements on
a code package. Typically, these take the form of an archive (e.g. .tgz or
.zip). The command will tell you where the package was saved so that you
can manually inspect or modify it.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if funcPackageCmdConfig.name == "source" {
			funcPackageCmdConfig.name = strings.TrimSuffix(path.Base(funcPackageCmdConfig.source), path.Ext(funcPackageCmdConfig.source))
		}

		includes := parseList(funcPackageCmdConfig.include)
		files := parseList(funcPackageCmdConfig.files)
		rawDir := srkManager.GetRawFunctionPath(funcPackageCmdConfig.name)

		if err := srkManager.CreateRawFunction(funcPackageCmdConfig.source, funcPackageCmdConfig.name, includes, files); err != nil {
			return errors.Wrap(err, "Packaging function failed")
		}
		srkManager.Logger.Info("Created raw function: " + rawDir)

		pkgPath, err := srkManager.Provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaging failed")
		}
		srkManager.Logger.Info("Package created at: " + pkgPath)
		return nil
	},
}

func init() {
	functionCmd.AddCommand(funcPackageCmd)

	// Define the command line arguments for this subcommand
	funcPackageCmd.Flags().StringVarP(&funcPackageCmdConfig.source, "source", "s", "", "source directory or file")
	funcPackageCmd.Flags().StringVarP(&funcPackageCmdConfig.include, "include", "i", "", "SRK-provided libraries to include")
	funcPackageCmd.Flags().StringVarP(&funcPackageCmdConfig.files, "files", "f", "", "additional files to include")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	funcPackageCmd.Flags().StringVarP(&funcPackageCmdConfig.name, "function-name", "n", "source", "Optional name for this function, if different than the source name")
}
