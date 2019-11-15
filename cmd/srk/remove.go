// The 'srk function remove' command. This uninstalls a function from the service.
package srk

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var removeName string

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Uninstall (remove) a function from the service provider.",
	Long:  `Remove will delete a function from the configured provider so that is no longer visible or using resources. If you have installed the function to multiple services, you will need to call "remove" on each service separately. Remove is the inverse of "install", it does not affect packages.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := srkConfig.provider.Faas.Remove(removeName); err != nil {
			return errors.Wrap(err, "Service removal failed")
		}

		srkConfig.logger.Info("Successfully removed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringVarP(&removeName, "function-name", "n", "", "The function to remove")
	removeCmd.MarkFlagRequired("function-name")
}
