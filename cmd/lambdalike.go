// Handle the "srk lambdalike" command
package cmd

import (
	"github.com/serverlessresearch/srk/pkg/lambdalike"
	"github.com/spf13/cobra"
)

var lambdaLikeCmd = &cobra.Command{
	Use:   "lambdalike",
	Short: "Administer LambdaLike FaaS implementation",
	Long:  `Run and administer LambdaLike functionality.`,
}

var lambdaLikeAPI = &cobra.Command{
	Use:   "apiservice",
	Short: "Run the LambdaLike API Service",
	RunE: func(cmd *cobra.Command, args []string) error {

		s := lambdalike.NewApiService([]string{}, 9001)
		s.Start()
		s.Wait()

		return nil
	},
}

var lambdaLikeWorker = &cobra.Command{
	Use:   "workermanager",
	Short: "Run the Worker Mananger",
	RunE: func(cmd *cobra.Command, args []string) error {

		wm := lambdalike.NewWorkerManager("", nil)
		wm.Shutdown()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lambdaLikeCmd)
	lambdaLikeCmd.AddCommand(lambdaLikeAPI)
	lambdaLikeCmd.AddCommand(lambdaLikeWorker)
}
