package cfbench

type ExperimentRunner struct {
}

type ConcurrencyScanRequest struct {
	FunctionName     string
	BeginConcurrency int
	EndConcurrency   int
	NumSteps         int
	StepDuration     int
}

type ExperimentRunResponse struct {
	Success bool
	Message string
}

func (er *ExperimentRunner) ConcurrencyScan(req ConcurrencyScanRequest, resp *ExperimentRunResponse) error {
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
