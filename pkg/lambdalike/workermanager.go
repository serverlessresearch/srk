package lambdalike

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
)

type WorkerManager struct {
	se           ServiceEndpoint
	tempDir      string
	maxFunctions int
	installed    []FunctionConfiguration
	instances    []*InstanceRunner
	wg           sync.WaitGroup
}

type FunctionConfiguration struct {
	FnName       string
	RuntimeAddr  string
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
	GetZipFile(name string) ([]byte, bool)
}

func NewWorkerManager(se ServiceEndpoint) *WorkerManager {
	tempDir, err := ioutil.TempDir("/tmp", "lambdalike")
	if err != nil {
		panic(err)
	}
	return &WorkerManager{
		tempDir: tempDir,
		se:      se,
	}
}

func (wm *WorkerManager) Shutdown() {
	for _, instance := range wm.instances {
		instance.Shutdown()
	}
	wm.wg.Wait()
}

// func (wm *WorkerManager) Configure(functions []lambda.FunctionConfiguration) error {

// 	for _, fc := range functions {
// 		codePath, err := wm.ensureCode(*fc.CodeSha256)
// 		if err != nil {
// 			return err
// 		}
// 		fi := wm.NewInstanceRunner(&fc, codePath)
// 		err = fi.Start()
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func (wm *WorkerManager) ConfigureOne(fc lambda.FunctionConfiguration, runtimeAddr string) error {
	codePath, err := wm.ensureCode(*fc.CodeSha256)
	if err != nil {
		return err
	}
	fi := wm.NewInstanceRunner(&fc, runtimeAddr, codePath)
	return fi.Start()
}

func (wm *WorkerManager) ensureCode(codeSha256 string) (string, error) {
	codePath := path.Join(wm.tempDir, codeSha256)
	info, err := os.Stat(codePath)
	if os.IsNotExist(err) {
		// Install
		code, found := wm.se.GetZipFile(codeSha256)
		if !found {
			return "", errors.Errorf("Unable to find code for %s", codeSha256)
		}
		codeBuf := bytes.NewReader(code)
		zipReader, err := zip.NewReader(codeBuf, int64(len(code)))
		if err != nil {
			return "", err
		}
		srk.ZipExpand(zipReader.File, codePath)
	} else {
		if err != nil {
			return "", errors.Wrap(err, "error checking code directory")
		}
		if !info.IsDir() {
			return "", errors.Errorf("not a directory at %s", codePath)
		}
	}
	return codePath, nil
}

type InstanceRunner struct {
	wm          *WorkerManager
	fc          *lambda.FunctionConfiguration
	runtimeAddr string
	sourcePath  string
	cmd         *exec.Cmd
	curState    string
	region      string
}

func (wm *WorkerManager) NewInstanceRunner(fc *lambda.FunctionConfiguration, runtimeAddr, sourcePath string) *InstanceRunner {
	ir := &InstanceRunner{wm, fc, runtimeAddr, sourcePath, nil, "", "us-west-2"}
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
		"-v", ir.sourcePath + ":/var/task",
		"--env", "_HANDLER=" + *ir.fc.Handler,
		"--env", "AWS_LAMBDA_FUNCTION_NAME=" + *ir.fc.FunctionName,
		"--env", "AWS_LAMBDA_FUNCTION_VERSION=" + *ir.fc.Version,
		"--env", "AWS_LAMBDA_FUNCTION_MEMORY_SIZE=" + strconv.FormatInt(*ir.fc.MemorySize, 10),
		"--env", "AWS_LAMBDA_LOG_GROUP_NAME=/aws/lambda/" + *ir.fc.FunctionName,
		"--env", "AWS_LAMBDA_LOG_STREAM_NAME='" + logStreamName(*ir.fc.Version) + "'",
		"--env", "AWS_REGION=" + ir.region,
		"--env", "AWS_DEFAULT_REGION=" + ir.region,
		"--env", "AWS_LAMBDA_RUNTIME_API=" + ir.runtimeAddr,
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
