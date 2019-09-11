// Handles the "srk function create" command

package cmd

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/serverlessresearch/srk/pkg/cfpackage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new function and register it with the FaaS provider",
	Long: `This will package up your function with any needed boilerplate and
upload it to the configured FaaS provider.`,
	Run: func(cmd *cobra.Command, args []string) {
		source := viper.GetString("source")
		includes := strings.Split(viper.GetString("include"), ",")
		target := "./build/functions/" +
			strings.TrimSuffix(path.Base(source), path.Ext(source)) +
			".zip"

		// Make the build directory if needed (users can clean the system by simply 'rm -r build')
		os.MkdirAll(filepath.Dir(target), os.ModePerm)

		if err := cfpackage.Package(source, target, includes); err != nil {
			panic(err)
		}
	},
}

func init() {
	functionCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("source", "s", "", "source directory or file")
	viper.BindPFlag("source", createCmd.Flags().Lookup("source"))

	createCmd.Flags().StringP("include", "i", "", "what to include, e.g., bench")
	viper.BindPFlag("include", createCmd.Flags().Lookup("include"))
}
