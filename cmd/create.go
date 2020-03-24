// Handles the "srk function create" command

package cmd

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var createCmdConfig struct {
	source  string
	include []string
	files   []string
	name    string
	env     map[string]string
	runtime string
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
		srkManager.Logger.Info("Function name: " + funcName)

		includes := createCmdConfig.include
		files := createCmdConfig.files
		env := createCmdConfig.env
		runtime := createCmdConfig.runtime
		rawDir := srkManager.GetRawPath(funcName)

		if err := srkManager.CreateRaw(createCmdConfig.source, funcName, includes, files); err != nil {
			return errors.Wrap(err, "Create command failed")
		}
		srkManager.Logger.Info("Created raw function: " + rawDir)

		pkgPath, err := srkManager.Provider.Faas.Package(rawDir)
		if err != nil {
			return errors.Wrap(err, "Packaging failed")
		}
		srkManager.Logger.Info("Created FaaS Package: " + pkgPath)

		if err := srkManager.Provider.Faas.Install(rawDir, env, runtime); err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkManager.Logger.Info("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(createCmd)

	// Define the command line arguments for this subcommand
	createCmd.Flags().StringVarP(&createCmdConfig.source, "source", "s", "", "source directory or file")
	createCmd.Flags().StringSliceVarP(&createCmdConfig.include, "include", "i", []string{}, "what to include, e.g., bench")
	createCmd.Flags().StringSliceVarP(&createCmdConfig.files, "files", "f", []string{}, "additional files to include")
	createCmd.Flags().StringToStringVarP(&createCmdConfig.env, "env", "e", make(map[string]string), "list of environment vars to set for function execution: var1=value1,var2=value2")
	createCmd.Flags().StringVarP(&createCmdConfig.runtime, "runtime", "r", "", "runtime to use for function execution")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	createCmd.Flags().StringVarP(&createCmdConfig.name, "function-name", "n", "source", "optional name for this function, if different than the source name")
}
