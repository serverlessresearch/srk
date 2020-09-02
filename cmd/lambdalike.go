// Handle the "srk lambdalike" command
package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/serverlessresearch/srk/pkg/lambdalike"
	"github.com/spf13/cobra"
)

var lambdaLikeCmd = &cobra.Command{
	Use:   "lambdalike",
	Short: "Administer LambdaLike FaaS implementation",
	Long:  `Run and administer LambdaLike functionality.`,
}

var lambdaLikeWorkerIPs []string
var lambdaLikeAPIAddr, lambdaLikeWorkerAddr string

var lambdaLikeAPI = &cobra.Command{
	Use:   "apiservice",
	Short: "Run the LambdaLike API Service",

	// Don't need the pre-run and post-run declared in root.go
	PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},

	RunE: func(cmd *cobra.Command, args []string) error {

		s := lambdalike.NewApiService(lambdaLikeWorkerIPs, lambdaLikeAPIAddr)
		err := s.Start()
		if err != nil {
			log.Fatal(err)
		}
		s.Wait()

		return nil
	},
}

var lambdaLikeWorker = &cobra.Command{
	Use:   "workermanager",
	Short: "Run the Worker Mananger",

	// Don't need the pre-run and post-run declared in root.go
	PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},

	RunE: func(cmd *cobra.Command, args []string) error {

		wm := lambdalike.NewWorkerManager(lambdaLikeWorkerAddr, nil)
		err := wm.Start()
		if err != nil {
			log.Fatal(err)
		}

		// Shutdown cleanly on ctrl-c or sigterm from kill
		done := make(chan struct{})
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
			close(done)
		}()
		<-done
		wm.Shutdown()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lambdaLikeCmd)

	lambdaLikeCmd.AddCommand(lambdaLikeAPI)
	lambdaLikeAPI.Flags().StringSliceVar(&lambdaLikeWorkerIPs, "workers", []string{}, "IP addresses of workers")
	lambdaLikeAPI.Flags().StringVar(&lambdaLikeAPIAddr, "address", "localhost:9001", "address for API service")

	lambdaLikeCmd.AddCommand(lambdaLikeWorker)
	lambdaLikeWorker.Flags().StringVar(&lambdaLikeWorkerAddr, "address", "localhost:8000", "address for Worker Service")
}
