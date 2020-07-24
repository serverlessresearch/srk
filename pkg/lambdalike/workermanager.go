package lambdalike

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"time"
)

type WorkerManager struct {
	se           ServiceEndpoint
	runtimeAddr  string
	maxFunctions int
	installed    []FunctionConfiguration
	instances    []*InstanceRunner
	wg           sync.WaitGroup
}

type FunctionConfiguration struct {
	FnName       string
	Version      string
	Handler      string
	MemSize      string
	Timeout      string
	Region       string // TODO move this out
	XAmznTraceID string // TODO - does this belong?
	Runtime      string
	ZipFileName  string
	NumInstances int
}

type ServiceEndpoint interface {
	GetZipFile(name string, cachedTag string) (bool, []byte)
}

func NewWorkerManager(runtimeAddr string, se ServiceEndpoint) *WorkerManager {
	log.Printf("creating network manager with address %s", runtimeAddr)
	return &WorkerManager{
		runtimeAddr: runtimeAddr,
	}
}

func (wm *WorkerManager) Shutdown() {
	for _, instance := range wm.instances {
		instance.Shutdown()
	}
	wm.wg.Wait()
}

func (wm *WorkerManager) Configure(functions []FunctionConfiguration) error {

	for _, fc := range functions {
		fi := wm.NewInstanceRunner(&fc, fc.ZipFileName)
		err := fi.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

type InstanceRunner struct {
	wm         *WorkerManager
	fc         *FunctionConfiguration
	sourcePath string
	cmd        *exec.Cmd
	curState   string
}

func (wm *WorkerManager) NewInstanceRunner(fc *FunctionConfiguration, sourcePath string) *InstanceRunner {
	ir := &InstanceRunner{wm, fc, sourcePath, nil, ""}
	wm.instances = append(wm.instances, ir)
	return ir
}

var bootstrapLaunchScript = `#!/bin/bash

set -e

for loc in "/var/runtime/bootstrap" "/var/task/bootstrap" "/opt/bootstrap"; do
    if [ -f $loc ]; then
        BOOTSTRAP="$loc"
        break
    fi
done

if [ -z "$BOOTSTRAP" ]; then
    echo "bootstrap not found"
    exit 1
fi

$BOOTSTRAP`

func (ir *InstanceRunner) Start() error {
	dockerArgs := []string{
		"run", "-i", "--rm",
		"--entrypoint", "/bin/bash",
		// "--env", "AWS_ACCESS_KEY_ID=" + awsAccessKey,
		// "--env", "AWS_SECRET_ACCESS_KEY=" + awsSecretKey,
		"-v", "/Users/jssmith/d/srk/examples/echo:/var/task",
		"--env", "_HANDLER=" + ir.fc.Handler,
		"--env", "AWS_LAMBDA_FUNCTION_NAME=" + ir.fc.FnName,
		"--env", "AWS_LAMBDA_FUNCTION_VERSION=" + ir.fc.Version,
		"--env", "AWS_LAMBDA_FUNCTION_MEMORY_SIZE=" + ir.fc.MemSize,
		"--env", "AWS_LAMBDA_LOG_GROUP_NAME=/aws/lambda/" + ir.fc.FnName,
		"--env", "AWS_LAMBDA_LOG_STREAM_NAME='" + logStreamName(ir.fc.Version) + "'",
		"--env", "AWS_REGION=" + ir.fc.Region,
		"--env", "AWS_DEFAULT_REGION" + ir.fc.Region,
		"--env", "_X_AMZN_TRACE_ID" + ir.fc.XAmznTraceID,
		"--env", "AWS_LAMBDA_RUNTIME_API=" + ir.wm.runtimeAddr,
		"lambci/lambda:python3.8",
	}

	ir.cmd = exec.Command("docker", dockerArgs...)
	fmt.Printf("Executing command %+v", ir.cmd)

	ir.cmd.Stdout = os.Stdout
	ir.cmd.Stderr = os.Stderr

	stdin, err := ir.cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := ir.cmd.Start(); err != nil {
		return err
	}
	_, err = io.WriteString(stdin, bootstrapLaunchScript)
	if err != nil {
		fmt.Printf("Failed to write command script %+v", err)
		// TODO how do we clean up properly + record that we are in a bad state?
		return err
	}
	err = stdin.Close()
	if err != nil {
		fmt.Printf("problem closing input stream")
		return err
	}

	ir.curState = "STATE_INIT"
	ir.wm.wg.Add(1)

	go func() {
		log.Printf("go awaiting docker termination")
		err := ir.cmd.Wait()
		if err != nil {
			fmt.Printf("wait on Docker terminated with %v", err)
			// TODO probably should have some retry with backoff here, e.g., could have an issue with the network
			// causing the failure.
		}
		// bootstrapIsRunning = false

		fmt.Printf("docker terminated with exit code %d\n", ir.cmd.ProcessState.ExitCode())
		ir.wm.wg.Done()
		// if !bootstrapExitedGracefully {
		// 	// context may have changed, use curContext instead
		// 	curContext.SetError(fmt.Errorf("Runtime exited without providing a reason"))
		// }

		// TODO restart if exited for some reason
	}()

	return nil
}

func (ir *InstanceRunner) Shutdown() {
	ir.cmd.Process.Kill()
}

func logStreamName(version string) string {
	randBuf := make([]byte, 16)
	rand.Read(randBuf)

	hexBuf := make([]byte, hex.EncodedLen(len(randBuf)))
	hex.Encode(hexBuf, randBuf)

	return time.Now().Format("2006/01/02") + "/[" + version + "]" + string(hexBuf)
}
