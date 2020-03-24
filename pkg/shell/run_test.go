package shell_test

import (
	"testing"

	"github.com/serverlessresearch/srk/pkg/shell"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {

	message := "hello, world"

	stdout, stderr, err := shell.Run(shell.Shell, shell.Exec, "/bin/echo -n "+message+" | tee /dev/stderr")
	assert.Nil(t, err)

	assert.Equal(t, message, string(stdout))
	assert.Equal(t, message, string(stderr))
}
