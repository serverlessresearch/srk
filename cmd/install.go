//Implements the 'srk function install' subcommand. Install takes a
//pre-packaged function and actually installs it to the function service.
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var installCmdConfig struct {
	name    string
	env     map[string]string
	runtime string
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a pre-packaged function to the configured FaaS service",
	Long:  `Install a function to the FaaS service. It is assumed that you have already packaged this function (using the 'package' command).`,
	RunE: func(cmd *cobra.Command, args []string) error {

		env := installCmdConfig.env
		runtime := installCmdConfig.runtime
		rawDir := srkManager.GetRawPath(installCmdConfig.name)

		if err := srkManager.Provider.Faas.Install(rawDir, env, runtime); err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkManager.Logger.Info("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installCmdConfig.name, "function-name", "n", "", "The function to install")
	installCmd.Flags().StringToStringVarP(&installCmdConfig.env, "env", "e", make(map[string]string), "list of environment vars to set for function execution: var1=value1,var2=value2")
	installCmd.Flags().StringVarP(&installCmdConfig.runtime, "runtime", "r", "", "runtime to use for function execution")
}
