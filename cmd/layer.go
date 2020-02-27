// Handles the "srk layer" command. This command exists solely to contain
// FaaS-specific subcommands (e.g. create, remove, etc..)

package cmd

import (
	"github.com/spf13/cobra"
)

// layerCmd represents the layer command
var layerCmd = &cobra.Command{
	Use:   "layer",
	Short: "Manage FaaS layer",
	Long:  `Commands for dealing with layers of your configured FaaS provider.`,
}

func init() {
	rootCmd.AddCommand(layerCmd)
}
