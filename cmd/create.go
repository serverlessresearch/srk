// Handles the "srk function create" command

package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/cobra"
)

var createCmdConfig struct {
	source  string
	include string
	service srk.FaasService
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new function and register it with the FaaS provider",
	Long: `This will package up your function with any needed boilerplate and
upload it to the configured FaaS provider. Create is equivalent to calling "srk
function package" and "srk function install".`,
	Run: func(cmd *cobra.Command, args []string) {

		createCmdConfig.service = getFaasService()
		defer createCmdConfig.service.Destroy()

		funcName := strings.TrimSuffix(path.Base(createCmdConfig.source), path.Ext(createCmdConfig.source))
		includes := strings.Split(createCmdConfig.include, ",")
		rawDir := getRawPath(funcName)

		if err := createRaw(createCmdConfig.source, funcName, includes, rawDir); err != nil {
			fmt.Println("Packaging function failed: %v\n", err)
			return
		}
		fmt.Println("Created raw function: " + rawDir)

		pkgPath, err := createCmdConfig.service.Package(rawDir)
		if err != nil {
			fmt.Printf("Packaing failed: %v\n", err)
			return
		}
		fmt.Println("Created FaaS Package: " + pkgPath)

		if err := createCmdConfig.service.Install(rawDir); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			return
		}
		fmt.Println("Successfully installed function")
	},
}

func init() {
	functionCmd.AddCommand(createCmd)

	// Define the command line arguments for this subcommand
	createCmd.Flags().StringVarP(&createCmdConfig.source, "source", "s", "", "source directory or file")
	createCmd.Flags().StringVarP(&createCmdConfig.include, "include", "i", "", "what to include, e.g., bench")
}
