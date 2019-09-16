/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installName string

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a pre-packaged function to the configured FaaS service",
	Long:  `Install a function to the FaaS service. It is assumed that you have already packaged this function (using the 'package' command).`,
	Run: func(cmd *cobra.Command, args []string) {
		service := getFaasService()
		defer service.Destroy()

		rawDir := getRawPath(installName)

		if err := service.Install(rawDir); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			return
		}
		fmt.Println("Successfully installed function")
	},
}

func init() {
	functionCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installName, "function-name", "n", "", "The function to install")
}
