// Handles the "srk function create" command

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	awslambda "github.com/serverlessresearch/srk/pkg/aws-lambda"
	"github.com/serverlessresearch/srk/pkg/openlambda"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
upload it to the configured FaaS provider.`,
	Run: func(cmd *cobra.Command, args []string) {

		configureCreate()

		funcName := strings.TrimSuffix(path.Base(createCmdConfig.source), path.Ext(createCmdConfig.source))
		includes := strings.Split(createCmdConfig.include, ",")

		rawDir, err := createRaw(createCmdConfig.source, funcName, includes)
		if err != nil {
			fmt.Println("Packaging function failed: %v\n", err)
			return
		}
		fmt.Println("Created raw function: " + rawDir)
		if err := createCmdConfig.service.Install(rawDir); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			return
		}
	},
}

// Runs only if the create command is invoked and after any cobra/viper setup
// has occured
func configureCreate() {
	// Setup the default FaaS service
	providerName := viper.GetString("default-provider")
	if providerName == "" {
		panic("No default provider in configuration")
	}

	serviceName := viper.GetString("providers." + providerName + ".faas")
	if serviceName == "" {
		panic("Provider \"" + providerName + "\" does not provide a FaaS service")
	}

	switch serviceName {
	case "openLambda":
		createCmdConfig.service = openlambda.NewConfig(
			viper.GetString("service.faas.openLambda.olcmd"),
			viper.GetString("service.faas.openLambda.oldir"))
	case "awsLambda":
		createCmdConfig.service = awslambda.NewConfig(
			viper.GetString("service.faas.awsLambda.role"),
			viper.GetString("service.faas.awsLambda.vpc-config"))
	default:
		panic("Unrecognized FaaS service: " + serviceName)
	}
}

func init() {
	functionCmd.AddCommand(createCmd)

	// Define the command line arguments for this subcommand
	createCmd.Flags().StringVarP(&createCmdConfig.source, "source", "s", "", "source directory or file")
	createCmd.Flags().StringVarP(&createCmdConfig.include, "include", "i", "", "what to include, e.g., bench")
}

// Place all provider-independent objects in a raw directory that will be
// packaged by the FaaS service.
func createRaw(source string, funcName string, includes []string) (rawDir string, err error) {
	//Shared global function build directory
	fBuildDir := filepath.Join(viper.GetString("buildDir"), "functions")
	err = os.MkdirAll(fBuildDir, os.ModeDir)
	if err != nil {
		fmt.Printf("Failed to create build directory at "+fBuildDir+": %v", err)
		return "", err
	}

	rawDir = filepath.Join(
		viper.GetString("buildDir"),
		"functions",
		funcName)

	// Always cleanup old raw directories first
	if err := os.RemoveAll(rawDir); err != nil {
		fmt.Printf("Failed to cleanup old build directory "+rawDir+": %v", err)
		return "", err
	}

	cmd := exec.Command("cp", "-r", source, rawDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Adding source returned error: %v\n", err)
		fmt.Printf(string(out))
		return rawDir, err
	}

	// Copy includes into the raw directory
	for _, include := range includes {
		includePath := filepath.Join(
			viper.GetString("includeDir"),
			"python",
			include)
		if _, err := os.Stat(includePath); err != nil {
			fmt.Printf("Couldn't find include: " + include)
			return rawDir, err
		}
		cmd := exec.Command("cp", "-r", includePath, rawDir)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Adding include returned error: %v", err)
			return rawDir, err
		}
	}

	return rawDir, nil
}
