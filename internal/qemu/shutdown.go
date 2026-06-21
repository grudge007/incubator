package qemu

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

func (q *QEMU) ShutdownVM(ctx context.Context, pid int) error {
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
		if !q.IsProcessRunning(pid) {
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

func (q *QEMU) IsProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return false
}
