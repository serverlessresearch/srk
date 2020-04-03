package shell

import (
	"errors"
	"io/ioutil"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

const (
	Shell = "/bin/bash"
	Exec  = "-c"
)

func Run(exe string, args ...string) ([]byte, []byte, error) {

	log.Debugf("exec %s %v", exe, args)

	cmd := exec.Command(exe, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	defer cmd.Wait()

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return nil, nil, err
	}

	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return nil, nil, err
	}

	return stdout, stderr, err
}

func RunSimple(exe string, args ...string) (string, error) {

	stdout, stderr, err := Run(exe, args...)
	if err != nil {
		return "", err
	}

	if len(stderr) > 0 {
		return "", errors.New(string(stderr))
	}

	return string(stdout), nil
}
