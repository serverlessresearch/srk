package lambcilambda

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"github.com/serverlessresearch/srk/pkg/command"
	"github.com/serverlessresearch/srk/pkg/srk"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	checkDelay = 100 * time.Millisecond
	maxChecks  = 10
)

type lambciRemote struct {
	scp  string // path to scp command
	ssh  string // path to ssh command
	host string // IP or hostname of server running the lambci/lambda docker image
	user string // user for scp + ssh
	pem  string // key file for scp + ssh
}

type lambciLambda struct {
	remote         *lambciRemote       // optional remote configuration
	address        string              // address of lambci server API
	envFile        string              // path to docker env file
	taskDir        string              // path to task directory
	runtimeDir     string              // path to runtime directory
	layersDir      string              // path to layer pool directory
	runtimes       map[string][]string // runtime configuration
	defaultRuntime string
	session        *lambda.Lambda
	log            srk.Logger
}

func NewFunctionService(logger srk.Logger, config *viper.Viper) (*lambciLambda, error) {

	var remote *lambciRemote
	if config.IsSet("remote") {
		remote = &lambciRemote{
			scp:  config.GetString("remote.scp"),
			ssh:  config.GetString("remote.ssh"),
			host: config.GetString("remote.host"),
			user: config.GetString("remote.user"),
			pem:  config.GetString("remote.pem"),
		}
	}

	service := &lambciLambda{
		remote:         remote,
		address:        config.GetString("address"),
		envFile:        config.GetString("env-file"),
		taskDir:        config.GetString("task-dir"),
		runtimeDir:     config.GetString("runtime-dir"),
		layersDir:      config.GetString("layers-dir"),
		runtimes:       make(map[string][]string),
		defaultRuntime: config.GetString("default-runtime"),
		session:        nil,
		log:            logger,
	}

	// not setting these values can have a bad outcome for the filesystem
	if service.envFile == "" {
		return nil, errors.New("configuration setting 'env-file' is required")
	}
	if service.taskDir == "" {
		return nil, errors.New("configuration setting 'task-dir' is required")
	}
	if service.runtimeDir == "" {
		return nil, errors.New("configuration setting 'runtime-dir' is required")
	}
	if service.layersDir == "" {
		return nil, errors.New("configuration setting 'layers-dir' is required")
	}

	for name, config := range config.GetStringMap("runtimes") {

		runtimeConfig := config.(map[string]interface{})
		layerConfig := runtimeConfig["layers"].([]interface{})

		layers := make([]string, len(layerConfig))
		for i := 0; i < len(layerConfig); i++ {
			layers[i] = layerConfig[i].(string)
		}

		service.runtimes[name] = layers
	}

	return service, nil
}

// Package up everything needed to install the function but don't actually
// install it to the service. rawDir may be assumed to be a unique path for
// this function. The package location should be determinsitically derived
// from the rawDir path.
// Returns: Path to the newly created package
func (service *lambciLambda) Package(rawDir string) (string, error) {

	// for lambci-lambda there is no need to create zip files
	return rawDir, nil
}

// Install a function to the desired FaaS service. It is assumed that
// Package() has already been called on this rawDir. The name of rawDir is
// also the name of the function.
func (service *lambciLambda) Install(rawDir string, env map[string]string, runtime string) error {

	if runtime == "" {
		if service.defaultRuntime == "" {
			return errors.New("runtime needs to be specified or configured via config")
		}
		runtime = service.defaultRuntime
	}

	if _, exists := service.runtimes[runtime]; !exists {
		return errors.Errorf("runtime '%s' does not exist in configuration", runtime)
	}

	// remove old layer
	_, err := service.Exec(fmt.Sprintf("find %s -mindepth 1 -maxdepth 1 -exec rm -r {} +", service.runtimeDir))
	if err != nil {
		return errors.Wrap(err, "error removing old layer")
	}

	// install new layer
	for _, layer := range service.runtimes[runtime] {
		_, err := service.Exec(fmt.Sprintf("cp -r %s %s", filepath.Join(service.layersDir, layer, "*"), service.runtimeDir))
		if err != nil {
			return errors.Wrapf(err, "error installing layer '%s'", layer)
		}
	}

	// remove old task
	_, err = service.Exec(fmt.Sprintf("find %s -mindepth 1 -maxdepth 1 -exec rm -r {} +", service.taskDir))
	if err != nil {
		return errors.Wrap(err, "error removing old task")
	}

	// install new task
	_, err = service.Copy(filepath.Join(rawDir, "*"), service.taskDir)
	if err != nil {
		return errors.Wrap(err, "error installing function")
	}

	// retrieve process id of running lambda docker image
	pid, err := service.Exec("ps ax | grep \"LAMBDA\" | grep -v entr | grep -v grep | awk \"{print $1}\"")
	if err != nil {
		return errors.Wrap(err, "error retrieving lambda process id")
	}

	// install new env map - this triggers the lambda function reload
	_, err = service.Exec(fmt.Sprintf("echo -n \"%s\" > %s", Map2Lines(env), service.envFile))
	if err != nil {
		return errors.Wrap(err, "error updating environment")
	}

	// wait until function reload happened in case of running lambda docker image
	if pid != "" {

		checks := 0
		for {
			time.Sleep(checkDelay)

			newPid, err := service.Exec("ps ax | grep \"LAMBDA\" | grep -v entr | grep -v grep | awk \"{print $1}\"")
			if err != nil {
				return errors.Wrap(err, "error retrieving lambda process id")
			}
			if newPid != "" && newPid != pid {
				break
			}

			if checks >= maxChecks {
				return errors.Errorf("lambda function container did not reload after %v, giving up", time.Duration(checks)*checkDelay)
			}
			checks++
		}
	}

	return nil
}

// Removes a function from the service. Does not affect packages.
func (service *lambciLambda) Remove(fName string) error {

	return nil
}

// Invoke function
// fName: Name of function
// args: JSON-encoded argument string
// Returns: function response as a bytes buffer. The exact format of this
// response may depend on the FaaS service. resp may be nil (indicating no
// valid response was received)
func (service *lambciLambda) Invoke(fName string, args string) (*bytes.Buffer, error) {

	url := fmt.Sprintf("http://%s/2015-03-31/functions/%s/invocations", service.address, fName)
	return HttpPost(url, args)
}

// Users must call Destroy on any created services to perform cleanup.
// Failure to destroy may leave the system in an inconsistent state that
// requires manual intervention.
func (service *lambciLambda) Destroy() {
	// nothing to do
}

// Report any collected statistics for this service. The collected
// statistics are dependent on the underlying implementation (you should
// always check if an expected category is available before reading).
func (service *lambciLambda) ReportStats() (map[string]float64, error) {
	// stats not implemented
	return nil, nil
}

// Resets all statistics to a 0 state. New calls to ReportStats() will only
// report new events.
func (service *lambciLambda) ResetStats() error {
	// stats not implemented
	return nil
}

func (service *lambciLambda) Exec(cmd string) (string, error) {

	if service.remote != nil {
		return command.Ssh(service.remote.ssh, service.remote.user, service.remote.host, service.remote.pem, cmd)
	} else {
		return command.Sh(cmd)
	}
}

func (service *lambciLambda) Copy(src, dst string) (string, error) {

	if service.remote != nil {
		return command.Scp(service.remote.scp, service.remote.user, service.remote.host, service.remote.pem, src, dst)
	} else {
		return command.Cp(src, dst)
	}
}
