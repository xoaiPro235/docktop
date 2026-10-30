//go:build windows

package docker

import (
	"os/exec"
	"strconv"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			// On Windows, TerminateProcess (cmd.Process.Kill()) only terminates the parent process (docker.exe).
			// Use taskkill /F /T /PID to recursively kill the entire process tree
			// (docker.exe and child compose processes), ensuring all inherited pipe handles are closed.
			killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
			if err := killCmd.Run(); err == nil {
				return nil
			}
			return cmd.Process.Kill()
		}
		return nil
	}
}
