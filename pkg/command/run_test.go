package command_test

import (
	"testing"

	"github.com/serverlessresearch/srk/pkg/command"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {

	message := "hello, world"

	stdout, stderr, err := command.Run(command.Shell, command.Exec, "/bin/echo -n "+message+" | tee /dev/stderr")
	assert.Nil(t, err)

	assert.Equal(t, message, string(stdout))
	assert.Equal(t, message, string(stderr))
}
