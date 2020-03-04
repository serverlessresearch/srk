package lambcilambda

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"
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

type lambciLambda struct {
	scp       string // path to scp command
	ssh       string // path to ssh command
	host      string // IP or hostname of server running the lambci/lambda docker image
	user      string // user for scp + ssh
	pem       string // key file for scp + ssh
	port      int    // port of the invoke API on server
	envFile   string // path to docker env file on server
	taskDir   string // path to task on server
	layerDir  string // path to layer on server
	layersDir string // path to layer pool on server
	session   *lambda.Lambda
	log       srk.Logger
}

func NewFunctionService(logger srk.Logger, config *viper.Viper) (*lambciLambda, error) {

	service := &lambciLambda{
		scp:       config.GetString("scp"),
		ssh:       config.GetString("ssh"),
		host:      config.GetString("host"),
		user:      config.GetString("user"),
		pem:       config.GetString("pem"),
		port:      config.GetInt("port"),
		envFile:   config.GetString("envFile"),
		taskDir:   config.GetString("taskDir"),
		layerDir:  config.GetString("layerDir"),
		layersDir: config.GetString("layersDir"),
		session:   nil,
		log:       logger,
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
func (service *lambciLambda) Install(rawDir string, env map[string]string, layers []string) error {

	// remove old layer
	_, err := service.Ssh(fmt.Sprintf("find %s -mindepth 1 -maxdepth 1 -exec rm -r {} +", service.layerDir))
	if err != nil {
		return errors.Wrap(err, "error removing old layer")
	}

	// install new layer
	for _, layer := range layers {
		_, err := service.Ssh(fmt.Sprintf("cp -r %s %s", filepath.Join(service.layersDir, layer, "*"), service.layerDir))
		if err != nil {
			return errors.Wrapf(err, "error installing layer '%s'", layer)
		}
	}

	// remove old task
	_, err = service.Ssh(fmt.Sprintf("find %s -mindepth 1 -maxdepth 1 -exec rm -r {} +", service.taskDir))
	if err != nil {
		return errors.Wrap(err, "error removing old task")
	}

	// install new task
	_, err = service.Scp(filepath.Join(rawDir, "*"), service.taskDir)
	if err != nil {
		return errors.Wrap(err, "error installing function")
	}

	// retrieve process id of running lambda docker image
	pid, err := service.Ssh("ps ax | grep \"LAMBDA\" | grep -v entr | grep -v grep | awk \"{print $1}\"")
	if err != nil {
		return errors.Wrap(err, "error retrieving lambda process id")
	}

	// install new env map - this triggers the lambda function reload
	_, err = service.Ssh(fmt.Sprintf("echo -n \"%s\" > %s", Map2Lines(env), service.envFile))
	if err != nil {
		return errors.Wrap(err, "error updating environment")
	}

	// wait until function reload happened in case of running lambda docker image
	if pid != "" {

		checks := 0
		for {
			time.Sleep(checkDelay)

			newPid, err := service.Ssh("ps ax | grep \"LAMBDA\" | grep -v entr | grep -v grep | awk \"{print $1}\"")
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

// Install a layer to the desired FaaS service. It is assumed that
// Package() has already been called on this rawDir. The name of rawDir is
// also the name of the layer.
// Returns: Id of the installed layer
func (service *lambciLambda) InstallLayer(rawDir string, compatibleRuntimes []string) (string, error) {

	// determine next version number
	layerName := path.Base(rawDir)
	layers, err := service.Ssh(fmt.Sprintf("find %s -maxdepth 1 -name \"%s*\" -type d", service.layersDir, layerName))
	if err != nil {
		return "", errors.Wrap(err, "error parsing existing layers")
	}

	// install new layer
	version := NextLayerVersion(strings.Split(layers, "\n"))
	layerId := fmt.Sprintf("%s-%d", layerName, version)
	_, err = service.Scp(rawDir, filepath.Join(service.layersDir, layerId))
	if err != nil {
		return "", errors.Wrapf(err, "error installing layer '%s'", layerId)
	}

	return layerId, nil
}

// Removes a layer from the service. Does not affect packages.
func (service *lambciLambda) RemoveLayer(name string) error {

	_, err := service.Ssh(fmt.Sprintf("find %s -maxdepth 1 -name \"%s*\" -type d -exec rm -r {} +", service.layersDir, name))
	if err != nil {
		return errors.Wrapf(err, "error removing layer '%s'", name)
	}

	return nil
}

// Invoke function
// fName: Name of function
// args: JSON-encoded argument string
// Returns: function response as a bytes buffer. The exact format of this
// response may depend on the FaaS service. resp may be nil (indicating no
// valid response was received)
func (service *lambciLambda) Invoke(fName string, args string) (*bytes.Buffer, error) {

	url := fmt.Sprintf("http://%s:%d/2015-03-31/functions/%s/invocations", service.host, service.port, fName)
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

func (service *lambciLambda) Ssh(cmd string) (string, error) {

	return command.Ssh(service.ssh, service.user, service.host, service.pem, fmt.Sprintf("'%s'", cmd))
}

func (service *lambciLambda) Scp(src, dst string) (string, error) {

	return command.Scp(service.scp, service.user, service.host, service.pem, src, dst)
}
