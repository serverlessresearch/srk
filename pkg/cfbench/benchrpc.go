package cfbench

import "time"

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
	var concurrencySpans []ConcurrencySpan
	totalDuration := time.Duration(req.LevelDuration*req.NumLevels) * time.Second
	for c := 0; c < req.NumLevels; c++ {
		concurrencySpans = append(concurrencySpans, ConcurrencySpan{
			concurrency: req.BeginConcurrency + (req.EndConcurrency-req.BeginConcurrency)/(req.NumLevels-1),
			begin:       time.Duration(c*req.LevelDuration) * time.Second,
			end:         totalDuration,
		})
	}
	//launchChannel := make(chan LaunchMessage)
	//completionChannel := make(chan CompletionMessage)
	cc, err := NewConcurrencyControl(concurrencySpans, er.LaunchChannel, er.CompletionChannel)
	if err != nil {
		return err
	}
	cc.Run()
	resp.Success = false
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
