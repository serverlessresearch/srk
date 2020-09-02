// Adapted from https://github.com/lambci/docker-lambda/blob/46ff80e2fe3bbb3fab4fa18ac2fb05d7167f064e/provided/run/init.go
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
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
)

type CodeServiceEndpoint interface {
	GetZipFile(name string) ([]byte, bool)
}

// WorkerManager manages the Docker containers that execute functions
type WorkerManager struct {
	configLock         sync.Mutex
	listenAddr         string
	defaultCodeService CodeServiceEndpoint
	tempDir            string
	maxFunctions       int
	runningInstances   map[string][]*InstanceRunner
	wg                 sync.WaitGroup
}

type WorkerManagerConfig struct {
	wm *WorkerManager
}

func (wmc *WorkerManagerConfig) Configure(req *ConfigRequest, resp *ConfigResponse) error {
	return wmc.wm.Configure(req, resp)
}

func NewWorkerManager(listenAddr string, se CodeServiceEndpoint) *WorkerManager {
	tempDir, err := ioutil.TempDir("/tmp", "lambdalike")
	if err != nil {
		panic(err)
	}
	return &WorkerManager{
		listenAddr:         listenAddr,
		tempDir:            tempDir,
		defaultCodeService: se,
		runningInstances:   make(map[string][]*InstanceRunner),
	}
}

func (wm *WorkerManager) Start() error {
	rpc.RegisterName("WorkerManager", &WorkerManagerConfig{wm})
	listener, err := net.Listen("tcp", wm.listenAddr)
	if err != nil {
		return err
	}
	log.Printf("Worker Manager listening at %s", listener.Addr().String())
	go rpc.Accept(listener)
	return nil
}

func (wm *WorkerManager) Shutdown() {
	for _, instanceList := range wm.runningInstances {
		for _, instance := range instanceList {
			instance.Shutdown()
		}
	}
	wm.wg.Wait()
}

type CodeServiceClient struct {
	// client *rpc.Client
	codeServiceAddr string
}

func NewCodeServiceClient(codeServiceAddr string) *CodeServiceClient {
	return &CodeServiceClient{codeServiceAddr}
}

func (c *CodeServiceClient) GetZipFile(name string) ([]byte, bool) {
	zipFileUrl := fmt.Sprintf("http://%s/zipfile/%s", c.codeServiceAddr, name)
	resp, err := http.Get(zipFileUrl)
	if err != nil {
		log.Printf("unable to get zip file for %s", zipFileUrl)
		return nil, false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}
	return body, true
}

type ConfigRequest struct {
	CodeServiceAddr              string
	WorkerFunctionConfigurations []WorkerFunctionConfiguration
}

type ConfigResponse struct{}

func (wm *WorkerManager) Configure(req *ConfigRequest, resp *ConfigResponse) error {
	log.Printf("Updating configuration")

	var err error
	var serviceEndpoint CodeServiceEndpoint

	wm.configLock.Lock()
	defer wm.configLock.Unlock()

	for _, c := range req.WorkerFunctionConfigurations {
		log.Printf("install configuration %q", c)
		configHash := c.hash()
		running, found := wm.runningInstances[configHash]
		runtimeAddr := c.RuntimeAddr
		var codePath string

		if !found || len(running) == 0 {
			log.Printf("ensuring code installed")
			if serviceEndpoint == nil {
				if req.CodeServiceAddr == "" {
					log.Printf("integrated code service")
					if wm.defaultCodeService == nil {
						return errors.New("missing code service endpoint")
					}
					serviceEndpoint = wm.defaultCodeService
				} else {
					log.Printf("fetching remote code from %s", req.CodeServiceAddr)
					serviceEndpoint = NewCodeServiceClient(req.CodeServiceAddr)
				}
			}
			// Don't have this running so download the code
			codePath, err = wm.ensureCode(*c.FunctionConfiguration.CodeSha256, serviceEndpoint)
			if err != nil {
				return err
			}
		} else {
			log.Printf("code already installed")
			runtimeAddr = running[0].runtimeAddr
			codePath = running[0].sourcePath
		}

		numRunning := len(running)
		if numRunning < c.NumInstances {
			log.Printf("starting up instances")
			numToLaunch := c.NumInstances - numRunning
			for i := 0; i < numToLaunch; i++ {
				ir := wm.NewInstanceRunner(c.FunctionConfiguration, runtimeAddr, codePath)
				ir.Start()
				running = append(running, ir)
			}
			wm.runningInstances[configHash] = running
		} else if numRunning > c.NumInstances {
			log.Printf("shutting down up instances")
			for _, instance := range running[c.NumInstances:] {
				instance.Shutdown()
			}
			if c.NumInstances == 0 {
				delete(wm.runningInstances, configHash)
			} else {
				wm.runningInstances[configHash] = running[:c.NumInstances]
			}
		}
	}

	log.Printf("Finished updating configuration")
	return nil
}

func (wm *WorkerManager) ensureCode(codeSha256 string, se CodeServiceEndpoint) (string, error) {
	log.Printf("getting code for %s", codeSha256)
	codePath := path.Join(wm.tempDir, codeSha256)
	info, err := os.Stat(codePath)
	if os.IsNotExist(err) {
		// Install
		code, found := se.GetZipFile(codeSha256)
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
	log.Printf("finished fetching code")
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
	runtimeAddr := strings.Replace(ir.runtimeAddr, "127.0.0.1", dockerHostIP, 1)
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
		"--env", "AWS_LAMBDA_RUNTIME_API=" + runtimeAddr,
	}

	if dockerHostNetworking {
		dockerArgs = append(dockerArgs, "--net=host")
	}

	gpuRequired := map[string]bool{
		"python3.8-cuda": true,
		"python3.8":      false,
	}

	addGPU, found := gpuRequired[*ir.fc.Runtime]
	if !found {
		return fmt.Errorf("Unkown runtime %s", ir.fc.Runtime)
	}
	if addGPU {
		dockerArgs = append(dockerArgs, "--gpus", "all")
	}
	dockerArgs = append(dockerArgs, fmt.Sprintf("lambci/lambda:%s", *ir.fc.Runtime))

	ir.cmd = exec.Command("docker", dockerArgs...)
	log.Printf("Executing command %+v", ir.cmd)

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
