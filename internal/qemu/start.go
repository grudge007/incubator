package qemu

import (
	"context"
	"os/exec"
)

func (q *QEMU) StartVM(ctx context.Context, qemuArgs []string) (int, error) {

	cmd := exec.Command(
		q.QemuBin,
		qemuArgs...,
	)
	err := cmd.Start()

	if err != nil {
		return 0, err
	}

	return cmd.Process.Pid, nil
}
