//Implements the 'srk function install' subcommand. Install takes a
//pre-packaged function and actually installs it to the function service.
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	installName string
	installEnv  string
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a pre-packaged function to the configured FaaS service",
	Long:  `Install a function to the FaaS service. It is assumed that you have already packaged this function (using the 'package' command).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rawDir := srkManager.GetRawPath(installName)

		if err := srkManager.Provider.Faas.Install(rawDir, parseKeyValue(installEnv)); err != nil {
			return errors.Wrap(err, "Installation failed")
		}
		srkManager.Logger.Info("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installName, "function-name", "n", "", "The function to install")
	installCmd.Flags().StringVarP(&installEnv, "env", "e", "", "list of environment vars: var1=value1,var2=value2")
}
