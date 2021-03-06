// Package creates a function archive that can be installed to a service.
package cmd

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var packageCmdConfig struct {
	source  string
	include []string
	files   []string
	name    string
}

// packageCmd represents the package command
var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package creates all the files needed to install a function, but does not actually install it.",
	Long: `Each FaaS service provider has their own format and requirements on
a code package. Typically, these take the form of an archive (e.g. .tgz or
.zip). The command will tell you where the package was saved so that you
can manually inspect or modify it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if packageCmdConfig.name == "source" {
			packageCmdConfig.name = strings.TrimSuffix(path.Base(packageCmdConfig.source), path.Ext(packageCmdConfig.source))
		}

		if packageCmdConfig.source, err = filepath.Abs(packageCmdConfig.source); err != nil {
			return err
		}

		rawDir := srkManager.GetRawPath(packageCmdConfig.name)

		if err := srkManager.CreateRaw(packageCmdConfig.source, packageCmdConfig.name, packageCmdConfig.include, packageCmdConfig.files); err != nil {
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
	functionCmd.AddCommand(packageCmd)

	// Define the command line arguments for this subcommand
	packageCmd.Flags().StringVarP(&packageCmdConfig.source, "source", "s", "", "source directory or file")
	packageCmd.Flags().StringSliceVarP(&packageCmdConfig.include, "include", "i", []string{}, "SRK-provided libraries to include")
	packageCmd.Flags().StringSliceVarP(&packageCmdConfig.files, "files", "f", []string{}, "additional files to include")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	packageCmd.Flags().StringVarP(&packageCmdConfig.name, "function-name", "n", "source", "optional name for this function, if different than the source name")
}
