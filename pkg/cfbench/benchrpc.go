package cfbench

import (
	"log"
	"time"
)

type ExperimentRunner struct {
	LaunchChannel     chan<- LaunchMessage
	CompletionChannel <-chan CompletionMessage
}

type ConcurrencyScanRequest struct {
	FunctionName     string
	BeginConcurrency int
	EndConcurrency   int
	NumLevels        int
	LevelDuration    int
}

type ExperimentRunResponse struct {
	Success bool
	Message string
}

func (er *ExperimentRunner) ConcurrencyScan(req ConcurrencyScanRequest, resp *ExperimentRunResponse) error {
	//var scanComplete = make(chan struct{})
	log.Printf("starting concurrency scan %+v", req)
	var (
		concurrencySpans []ConcurrencySpan
		concurrency      int
		totalDuration    = time.Duration(req.LevelDuration*req.NumLevels) * time.Second
		stepConcurrency  = (req.EndConcurrency - req.BeginConcurrency) / (req.NumLevels - 1)
	)
	for c := 0; c < req.NumLevels; c++ {
		if c == 0 {
			concurrency = req.BeginConcurrency
		} else {
			concurrency = stepConcurrency
		}
		concurrencySpans = append(concurrencySpans, ConcurrencySpan{
			concurrency: concurrency,
			begin:       time.Duration(c*req.LevelDuration) * time.Second,
			end:         totalDuration,
		})
	}

	cc, err := NewConcurrencyControl(concurrencySpans, er.LaunchChannel, er.CompletionChannel)
	if err != nil {
		return err
	}
	cc.Run()
	resp.Success = true
	return nil
}

type StatusRequest struct {
}

type StatusResponse struct {
	Ok bool
}

func (er *ExperimentRunner) Status(req StatusRequest, resp *StatusResponse) error {
	resp.Ok = true
	return nil
}
