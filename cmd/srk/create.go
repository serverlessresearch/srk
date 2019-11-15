// Handles the "srk function create" command

package srk

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var createCmdConfig struct {
	source  string
	include string
	name    string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new function and register it with the FaaS provider",
	Long: `This will package up your function with any needed boilerplate and
upload it to the configured FaaS provider. Create is equivalent to calling "srk
function package" and "srk function install".`,
	RunE: func(cmd *cobra.Command, args []string) error {

		var funcName string
		if createCmdConfig.name == "source" {
			funcName = strings.TrimSuffix(path.Base(createCmdConfig.source), path.Ext(createCmdConfig.source))
		} else {
			funcName = createCmdConfig.name
		}
		srkConfig.logger.Info("Function name: " + funcName)

		includes := strings.Split(createCmdConfig.include, ",")
		rawDir := getRawPath(funcName)

		if err := createRaw(createCmdConfig.source, funcName, includes, rawDir); err != nil {
			return errors.Wrap(err, "Create command failed")
		}
		srkConfig.logger.Info("Created raw function: " + rawDir)

		// pkgPath, err := createCmdConfig.service.Package(rawDir)
		pkgPath, err := srkConfig.provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaging failed")
		}
		srkConfig.logger.Info("Created FaaS Package: " + pkgPath)

		// if err := createCmdConfig.service.Install(rawDir); err != nil {
		if err := srkConfig.provider.Faas.Install(rawDir); err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkConfig.logger.Info("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(createCmd)

	// Define the command line arguments for this subcommand
	createCmd.Flags().StringVarP(&createCmdConfig.source, "source", "s", "", "source directory or file")
	createCmd.Flags().StringVarP(&createCmdConfig.include, "include", "i", "", "what to include, e.g., bench")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	createCmd.Flags().StringVarP(&createCmdConfig.name, "function-name", "n", "source", "Optional name for this function, if different than the source name")
}
