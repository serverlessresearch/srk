// Standard interfaces and datatypes for the SRK project.
// Terms:
//   "service" : A specific implementation of some cloud functionality (e.g. compute, key-value store, etc.)
//   "provider" : A coherent set of services that all work together simultaneously
package srk

type FaasService interface {
	// Install a function to the desired FaaS service
	// inputPath - path to the input function source
	// funcName - FaaS service visible name for the function
	// includes - Any extra boilerplate to include (defined in cfincludes)
	Install(rawDir string) (rerr error)
}

// A provider aggregates a set of services that all run simultaneously. In
// theory, you can mix-and-match, but in practice only certain combinations may
// work.
type Provider struct {
	faas *FaasService
}

// Benchmarks
type IncludedFile struct {
	name    string
	content func() []byte
}

type Benchmark interface {
	GetIncludes() *[]IncludedFile
	// XXX TODO
	RunBench() error
}
