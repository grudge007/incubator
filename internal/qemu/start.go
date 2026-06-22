package qemu

import (
	"context"
	"fmt"
	"incubator/internal/model"
	"os/exec"
	"strconv"
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

func (q *QEMU) PrepareQemuArgs(vmDetails model.VM) []string {
	memory := strconv.Itoa(vmDetails.MemoryMB)
	cpu := strconv.Itoa(vmDetails.CPUs)
	driveArg := genDriveArg(vmDetails.BootDisk)
	cloudInitArg := fmt.Sprintf("file=%s,format=raw,media=cdrom", vmDetails.CloudInitFile)
	vncArg := fmt.Sprintf("0.0.0.0:%d", vmDetails.VNC)

	qemuArgs := []string{
		"-enable-kvm",
		"-m", memory,
		"-name", vmDetails.Name,
		"-smp", cpu,
		"-cpu", "host",
		"-drive", driveArg,
		"-drive", cloudInitArg,
		"-display", "none",
		"-vnc", vncArg,
	}
	return qemuArgs
}
