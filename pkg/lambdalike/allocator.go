package lambdalike

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"log"
	"net/rpc"

	"github.com/pkg/errors"
)

// Allocator is resposible for resource allocation
type Allocator struct {
	codeServiceAddr string
	functions       map[string]WorkerFunctionConfiguration
	workers         []string
}

func NewAllocator(workers []string, codeServiceAddr string) *Allocator {
	return &Allocator{
		functions:       make(map[string]WorkerFunctionConfiguration),
		workers:         workers,
		codeServiceAddr: codeServiceAddr,
	}
}

func (a *Allocator) AddFunction(c WorkerFunctionConfiguration) error {
	a.functions[c.hash()] = c
	return a.update()
}

func (a *Allocator) update() error {
	log.Printf("allocator update starting")
	// We hash to a starting point, then allocate instances sequentially
	numWorkers := len(a.workers)
	configs := make([][]WorkerFunctionConfiguration, numWorkers)

	for hash, fc := range a.functions {
		var n uint64
		sha256Bytes, err := base64.URLEncoding.DecodeString(hash)
		binary.Read(bytes.NewReader(sha256Bytes), binary.LittleEndian, &n)
		if err != nil {
			return err
		}
		startIndex := int(n % uint64(numWorkers))
		remaining := fc.NumInstances
		incr := (fc.NumInstances-1)/numWorkers + 1
		for i := 0; i < fc.NumInstances; i++ {
			workerInstances := incr
			if workerInstances < remaining {
				workerInstances = remaining
			}
			index := (startIndex + i) % numWorkers
			configs[index] = append(configs[index],
				WorkerFunctionConfiguration{
					FunctionConfiguration: fc.FunctionConfiguration,
					RuntimeAddr:           fc.RuntimeAddr,
					NumInstances:          workerInstances,
				})
			remaining -= workerInstances
		}
	}

	for i := 0; i < numWorkers; i++ {
		log.Printf("configure %d functions on %s", len(configs[i]), a.workers[i])
		client, err := rpc.Dial("tcp", a.workers[i])
		if err != nil {
			return err
		}
		defer client.Close()
		req := ConfigRequest{
			CodeServiceAddr:              a.codeServiceAddr,
			WorkerFunctionConfigurations: configs[i],
		}
		var resp ConfigResponse
		err = client.Call("WorkerManager.Configure", &req, &resp)
		if err != nil {
			return errors.Wrapf(err, "failed to update worker %s", a.workers[i])
		}
	}
	log.Printf("allocator update finished")
	return nil
}
