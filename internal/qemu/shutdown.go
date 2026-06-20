package qemu

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

func (q *QEMU) ShutdownVM(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) || err.Error() == "os: process already finished" {
			return nil
		}
		return err
	}

	for range 5 {
		if !isProcessRunning(proc) {
			fmt.Println("Succesfully Stopped Resource")
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	if err = proc.Signal(syscall.SIGKILL); err != nil {
		return err
	}

	return nil
}

func isProcessRunning(proc *os.Process) bool {
	err := proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return false

}
