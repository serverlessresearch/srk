// Standard interfaces and datatypes for the SRK project.
// Terms:
//   "service" : A specific implementation of some cloud functionality (e.g. compute, key-value store, etc.)
//   "provider" : A coherent set of services that all work together simultaneously
package srk

import "bytes"

// A provider aggregates a set of services that all run simultaneously. In
// theory, you can mix-and-match, but in practice only certain combinations may
// work.
type Provider struct {
	Faas FunctionService
}

type FunctionService interface {
	// Package up everything needed to install the function but don't actually
	// install it to the service. rawDir may be assumed to be a unique path for
	// this function. The package location should be determinsitically derived
	// from the rawDir path.
	// Returns: Path to the newly created package
	Package(rawDir string) (pkgPath string, rerr error)

	// Install a function to the desired FaaS service. It is assumed that
	// Package() has already been called on this rawDir.
	Install(rawDir string) (rerr error)

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
}

// Benchmarks
type BenchArgs struct {
	FName       string
	FArgs       string
	BParams     string
	TrackingUrl string
	Output      string
}

type Benchmark interface {
	RunBench(prov *Provider, args *BenchArgs) error
}

// Every distinct benchmark should provide a factory
type BenchFactory func() (Benchmark, error)
