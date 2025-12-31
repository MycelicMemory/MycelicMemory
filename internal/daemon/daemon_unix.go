//go:build !windows
// +build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

// setProcAttr sets the process attributes for daemonization on Unix systems
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
