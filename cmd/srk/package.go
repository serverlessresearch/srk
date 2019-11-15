// Package creates a function archive that can be installed to a service.
package srk

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var packageCmdConfig struct {
	source  string
	include string
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

		if packageCmdConfig.name == "source" {
			packageCmdConfig.name = strings.TrimSuffix(path.Base(packageCmdConfig.source), path.Ext(packageCmdConfig.source))
		}

		includes := strings.Split(packageCmdConfig.include, ",")
		rawDir := getRawPath(packageCmdConfig.name)

		if err := createRaw(packageCmdConfig.source, packageCmdConfig.name, includes, rawDir); err != nil {
			return errors.Wrap(err, "Packaging function failed")
		}
		srkConfig.logger.Info("Created raw function: " + rawDir)

		pkgPath, err := srkConfig.provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaing failed")
		}
		srkConfig.logger.Info("Package created at: " + pkgPath)
		return nil
	},
}

func init() {
	functionCmd.AddCommand(packageCmd)

	// Define the command line arguments for this subcommand
	packageCmd.Flags().StringVarP(&packageCmdConfig.source, "source", "s", "", "source directory or file")
	packageCmd.Flags().StringVarP(&packageCmdConfig.include, "include", "i", "", "SRK-provided libraries to include")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	packageCmd.Flags().StringVarP(&packageCmdConfig.name, "function-name", "n", "source", "Optional name for this function, if different than the source name")

}

// Returns a path to the raw directory for funcName (whether it exists or not)
func getRawPath(funcName string) string {
	return filepath.Join(
		viper.GetString("buildDir"),
		"functions",
		funcName)
}

// Place all provider-independent objects in a raw directory that will be
// packaged by the FaaS service. Will replace any existing rawDir.
// source: is the path to the user-provided source directory
// funcName: Unique name to give this function
// includes: List of standard SRK libraries to include (just the names of the packages, not paths)
// rawDir: Path where the rawDir should be made
func createRaw(source string, funcName string, includes []string, rawDir string) (err error) {
	//Shared global function build directory
	fBuildDir := filepath.Join(viper.GetString("buildDir"), "functions")
	err = os.MkdirAll(fBuildDir, 0775)
	if err != nil {
		return errors.Wrap(err, "Failed to create build directory at "+fBuildDir)
	}

	// Always cleanup old raw directories first
	if err := os.RemoveAll(rawDir); err != nil {
		return errors.Wrap(err, "Failed to cleanup old build directory "+rawDir)
	}

	cmd := exec.Command("cp", "-r", source, rawDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "Adding source returned error\n%v", out)
	}

	// Copy includes into the raw directory
	for _, include := range includes {
		includePath := filepath.Join(
			viper.GetString("includeDir"),
			"python",
			include)
		if _, err := os.Stat(includePath); err != nil {
			return errors.Wrap(err, "Couldn't find include: "+include)
		}
		cmd := exec.Command("cp", "-r", includePath, rawDir)
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "Adding include returned error")
		}
	}

	return nil
}
