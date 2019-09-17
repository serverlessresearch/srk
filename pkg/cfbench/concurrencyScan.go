// The concurrency scan benchmark (implements the Benchmark interface). This
// benchmark measures the scaling and concurrency under load of a FaaS service.
package cfbench

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Implements the Benchmark interface
type ConcurrencySweepBench struct {
	params string
}

type ConcurrencySweepArgs struct {
	Begin        int `json:"begin_concurrency"`
	Delta        int `json:"delta_concurrency"`
	Steps        int `json:"num_steps"`
	StepDuration int `json:"step_duration"`
}

type TransitionPoint struct {
	concurrency int
	when        time.Duration
}

func NewConcurrencySweepBench(mode string, params string) *ConcurrencySweepBench {
	return &ConcurrencySweepBench{
		params: params,
	}
}

func GenSweepTransitions(args ConcurrencySweepArgs) *[]TransitionPoint {
	transitions := make([]TransitionPoint, 0, args.Steps+1)
	for step := 0; step < args.Steps; step++ {
		transitions = append(transitions,
			TransitionPoint{
				args.Begin + step*args.Delta,
				time.Duration(step*args.StepDuration) * time.Second,
			})
	}
	transitions = append(transitions, TransitionPoint{0, time.Duration(args.Steps*args.StepDuration) * time.Second})
	return &transitions
}

func ConcurrencySweep(functionName string, functionArgs map[string]interface{}, sweepDefinition *[]TransitionPoint, trackingUrl string, logfile string) {
	experimentId := genExperimentId()
	log.Printf("starting experiment %s", experimentId)
	progress := newProgress(experimentId)

	serverWorking := make(chan struct{})

	// Create a log writer
	logWriter := make(chan string)
	logWriterWorking := make(chan struct{})
	go func() {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
		for s := range logWriter {
			_, err = fmt.Fprintf(f, "%s\n", s)
			if err != nil {
				log.Printf("Unable to log %v", err)
			}
		}
		log.Printf("experiment data saved to %s", logfile)
		close(logWriterWorking)
	}()

	// Start the server and invoke function execution
	go ExperimentServer(&progress, logWriter, serverWorking)
	go invokeMulti(experimentId, trackingUrl, functionName, functionArgs, sweepDefinition, &progress)

	<-serverWorking

	close(logWriter)
	<-logWriterWorking
}
