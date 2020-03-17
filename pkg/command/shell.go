package command

import (
	"fmt"
)

func Cp(src, dst string) (string, error) {

	cmd := fmt.Sprintf("cp -a %s %s", src, dst)
	return RunSimple(Shell, Exec, cmd)
}

func Sh(cmd string) (string, error) {

	return RunSimple(Shell, Exec, cmd)
}
