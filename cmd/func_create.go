// Handles the "srk function create" command

package cmd

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var funcCreateCmdConfig struct {
	source  string
	include string
	files   string
	name    string
	env     string
	runtime string
}

var funcCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new function and register it with the FaaS provider",
	Long: `This will package up your function with any needed boilerplate and
upload it to the configured FaaS provider. Create is equivalent to calling "srk
function package" and "srk function install".`,
	RunE: func(cmd *cobra.Command, args []string) error {

		var funcName string
		if funcCreateCmdConfig.name == "source" {
			funcName = strings.TrimSuffix(path.Base(funcCreateCmdConfig.source), path.Ext(funcCreateCmdConfig.source))
		} else {
			funcName = funcCreateCmdConfig.name
		}
		srkManager.Logger.Info("Function name: " + funcName)

		includes := parseList(funcCreateCmdConfig.include)
		files := parseList(funcCreateCmdConfig.files)
		runtime := funcCreateCmdConfig.runtime
		rawDir := srkManager.GetRawFunctionPath(funcName)

		if err := srkManager.CreateRawFunction(funcCreateCmdConfig.source, funcName, includes, files); err != nil {
			return errors.Wrap(err, "Create command failed")
		}
		srkManager.Logger.Info("Created raw function: " + rawDir)

		pkgPath, err := srkManager.Provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaging failed")
		}
		srkManager.Logger.Info("Created FaaS Package: " + pkgPath)

		if err := srkManager.Provider.Faas.Install(rawDir, parseKeyValue(funcCreateCmdConfig.env), runtime); err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkManager.Logger.Info("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(funcCreateCmd)

	// Define the command line arguments for this subcommand
	funcCreateCmd.Flags().StringVarP(&funcCreateCmdConfig.source, "source", "s", "", "source directory or file")
	funcCreateCmd.Flags().StringVarP(&funcCreateCmdConfig.include, "include", "i", "", "what to include, e.g., bench")
	funcCreateCmd.Flags().StringVarP(&funcCreateCmdConfig.files, "files", "f", "", "additional files to include")
	funcCreateCmd.Flags().StringVarP(&funcCreateCmdConfig.env, "env", "e", "", "list of environment vars: var1=value1,var2=value2")
	funcCreateCmd.Flags().StringVarP(&funcCreateCmdConfig.runtime, "runtime", "r", "", "runtime to use for function execution")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	funcCreateCmd.Flags().StringVarP(&funcCreateCmdConfig.name, "function-name", "n", "source", "optional name for this function, if different than the source name")
}
