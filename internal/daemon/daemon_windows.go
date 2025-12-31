//go:build windows
// +build windows

package daemon

import (
	"os/exec"
	"syscall"
)

// setProcAttr sets the process attributes for daemonization on Windows
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
