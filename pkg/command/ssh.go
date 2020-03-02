package command

import (
	"fmt"
)

func Scp(exe, username, hostname, pem, src, dst string) (string, error) {

	cmd := fmt.Sprintf("%s -r -C -i %s %s %s@%s:%s", exe, pem, src, username, hostname, dst)
	return RunSimple(Shell, Exec, cmd)
}

func Ssh(exe, username, hostname, pem, command string) (string, error) {

	cmd := fmt.Sprintf("%s -i %s %s@%s %s", exe, pem, username, hostname, command)
	return RunSimple(Shell, Exec, cmd)
}
