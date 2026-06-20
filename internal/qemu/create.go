package qemu

import (
	"fmt"
	"incubator/internal/model"
	"os/exec"
	"strconv"
)

type Build struct {
}

func (q *QEMU) CreateVM(vmDetails *model.VM, imagePath string) (*model.VM, error) {

	memory := strconv.Itoa(vmDetails.MemoryMB)
	cpu := strconv.Itoa(vmDetails.CPUs)
	resourceId := strconv.Itoa(vmDetails.ResourceID)
	diskDir := genDiskDir(q.DiskStore, resourceId)
	fmt.Println("%#W", diskDir)
	diskPath := genDiskPath(q.DiskStore, resourceId)
	driveArg := genDriveArg(diskPath)
	err := createVMDisk(imagePath, diskPath, resourceId, q.DiskStore)
	if err != nil {
		return vmDetails, err
	}

	err = resizeVMDisk(diskPath, vmDetails.DiskSize)
	if err != nil {
		return vmDetails, err
	}

	cloudInitDisk, err := createCloudInitIso(q.CloudInitFile, diskDir)
	if err != nil {
		return vmDetails, err
	}
	cloudInitArg := fmt.Sprintf("file=%s,format=raw,media=cdrom", cloudInitDisk)
	fmt.Printf("\nVNC FROM QEMU: %d\n", vmDetails.VNC)

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
	fmt.Printf("cmd : %v", cmd)
	err = cmd.Start()
	if err != nil {
		return vmDetails, err
	}
	vmDetails.PID = cmd.Process.Pid

	fmt.Println("PID: ", vmDetails.PID)
	vm := *vmDetails
	vm.DiskIndex = 0
	vm.BootDisk = diskPath
	vm.CloudInitFile = cloudInitDisk
	vm.Status = "running"
	vm.DiskType = "boot"
	return &vm, nil
}
