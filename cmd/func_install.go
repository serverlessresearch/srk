//Implements the 'srk function install' subcommand. Install takes a
//pre-packaged function and actually installs it to the function service.
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var funcInstallCmdConfig struct {
	name   string
	env    string
	layers string
}

// funcInstallCmd represents the install command
var funcInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a pre-packaged function to the configured FaaS service",
	Long:  `Install a function to the FaaS service. It is assumed that you have already packaged this function (using the 'package' command).`,
	RunE: func(cmd *cobra.Command, args []string) error {

		layers := parseList(funcInstallCmdConfig.layers)
		rawDir := srkManager.GetRawFunctionPath(funcInstallCmdConfig.name)

		if err := srkManager.Provider.Faas.Install(rawDir, parseKeyValue(funcInstallCmdConfig.env), layers); err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkManager.Logger.Info("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(funcInstallCmd)

	funcInstallCmd.Flags().StringVarP(&funcInstallCmdConfig.name, "function-name", "n", "", "The function to install")
	funcInstallCmd.Flags().StringVarP(&funcInstallCmdConfig.env, "env", "e", "", "list of environment vars: var1=value1,var2=value2")
	funcInstallCmd.Flags().StringVarP(&funcInstallCmdConfig.layers, "layers", "l", "", "list of additional layer ARNs")
}
