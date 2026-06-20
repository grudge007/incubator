package qemu

import (
	"fmt"
	"incubator/internal/model"
	"os/exec"
	"strconv"
)

func (q *QEMU) StartVM(vmDetails model.VM) (int, error) {
	memory := strconv.Itoa(vmDetails.MemoryMB)
	cpu := strconv.Itoa(vmDetails.CPUs)
	driveArg := genDriveArg(vmDetails.BootDisk)
	cloudInitArg := fmt.Sprintf("file=%s,format=raw,media=cdrom", vmDetails.CloudInitFile)
	vncArg := fmt.Sprintf("0.0.0.0:%d", vmDetails.VNC)

	cmd := exec.Command(
		q.QemuBin,
		"-enable-kvm",
		"-m", memory,
		"-name", vmDetails.Name,
		"-smp", cpu,
		"-cpu", "host",
		"-drive", driveArg,
		"-drive", cloudInitArg,
		"-display", "none",
		"-vnc", vncArg,
	)
	err := cmd.Start()

	if err != nil {
		return 0, err
	}

	return cmd.Process.Pid, nil
}
