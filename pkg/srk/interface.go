// Standard interfaces and datatypes for the SRK project.
// Terms:
//   "service" : A specific implementation of some cloud functionality (e.g. compute, key-value store, etc.)
//   "provider" : A coherent set of services that all work together simultaneously
package srk

import (
	"bytes"

	"github.com/sirupsen/logrus"
)

// A provider aggregates a set of services that all run simultaneously. In
// theory, you can mix-and-match, but in practice only certain combinations may
// work.
type Provider struct {
	Faas FunctionService
}

// A function service provides a FaaS interface
// All new function services should provide an object that meets this interface, with a constructor like:
// func NewConfig(logger Logger, config *viper.Viper) (FunctionService, error)
type FunctionService interface {

	// Package up everything needed to install the function but don't actually
	// install it to the service. rawDir may be assumed to be a unique path for
	// this function. The package location should be determinsitically derived
	// from the rawDir path.
	// Returns: Path to the newly created package
	Package(rawDir string) (pkgPath string, rerr error)

	// Install a function to the desired FaaS service. It is assumed that
	// Package() has already been called on this rawDir. The name of rawDir is
	// also the name of the function.
	Install(rawDir string, env map[string]string, layers []string) (rerr error)

	// Removes a function from the service. Does not affect packages.
	Remove(fName string) (rerr error)

	// Invoke function
	// fName: Name of function
	// args: JSON-encoded argument string
	// Returns: function response as a bytes buffer. The exact format of this
	// response may depend on the FaaS service. resp may be nil (indicating no
	// valid response was received)
	Invoke(fName string, args string) (resp *bytes.Buffer, rerr error)

	// Users must call Destroy on any created services to perform cleanup.
	// Failure to destroy may leave the system in an inconsistent state that
	// requires manual intervention.
	Destroy()

	// Report any collected statistics for this service. The collected
	// statistics are dependent on the underlying implementation (you should
	// always check if an expected category is available before reading).
	ReportStats() (map[string]float64, error)

	// Resets all statistics to a 0 state. New calls to ReportStats() will only
	// report new events.
	ResetStats() error
}

type BenchArgs struct {
	FName       string
	FArgs       string
	BParams     string
	TrackingUrl string
	Output      string
}

// Benchmarks use a provider to run some experiment. They can install
// functions, access data stores, and invoke services as needed.
// They should provide a constructor like:
// func NewBench(logger Logger) (Benchmark, error)
type Benchmark interface {
	RunBench(prov *Provider, args *BenchArgs) error
}

// Alias logrus FieldLogger in case we want to change the logging behavior in
// the future (e.g. add methods to the interface)
type Logger logrus.FieldLogger
