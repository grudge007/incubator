package qemu

import (
	"context"
	"fmt"
	"incubator/internal/logger"
	"incubator/internal/model"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

type Build struct {
}

func (q *QEMU) PrepareVMCreation(ctx context.Context, vmDetails *model.VM, imagePath string) (*model.VM, []string, error) {

	memory := strconv.Itoa(vmDetails.MemoryMB)
	cpu := strconv.Itoa(vmDetails.CPUs)
	resourceId := strconv.Itoa(vmDetails.ResourceID)
	diskDir := genDiskDir(q.DiskStore, resourceId)
	fmt.Println("%#W", diskDir)
	diskPath := genDiskPath(q.DiskStore, resourceId)
	driveArg := genDriveArg(diskPath)

	err := createVMDisk(imagePath, diskPath, resourceId, q.DiskStore)
	if err != nil {
		logger.LogError(ctx, vmDetails.ResourceID, "create-vm", "failed to create vm disk", err)
		return vmDetails, nil, err
	}
	logger.LogSuccess(ctx, vmDetails.ResourceID, "create-vm", "succesfully created vm disk", imagePath)

	err = resizeVMDisk(diskPath, vmDetails.DiskSize)
	if err != nil {
		logger.LogError(ctx, vmDetails.ResourceID, "create-vm", "failed to resize vm disk", err)
		return vmDetails, nil, err
	}

	logger.LogSuccess(ctx, vmDetails.ResourceID, "create-vm", "succesfully resized vm disk", strconv.Itoa(vmDetails.DiskSize))

	cloudInitDisk, err := createCloudInitIso(q.CloudInitFile, diskDir)
	if err != nil {
		logger.LogError(ctx, vmDetails.ResourceID, "create-vm", "failed to generate cloud init disk", err)
		return vmDetails, nil, err
	}

	cloudInitArg := fmt.Sprintf("file=%s,format=raw,media=cdrom", cloudInitDisk)
	vncArg := fmt.Sprintf("0.0.0.0:%d", vmDetails.VNC)

	vm := *vmDetails
	vm.DiskIndex = 0
	vm.BootDisk = diskPath
	vm.CloudInitFile = cloudInitDisk
	vm.DiskType = "boot"

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

	return &vm, qemuArgs, nil
}

// logger.LogSuccess(ctx, vmDetails.ResourceID, "create-vm", "succesfully created cloud-init disk", cloudInitDisk)

// cloudInitArg := fmt.Sprintf("file=%s,format=raw,media=cdrom", cloudInitDisk)

// vncArg := fmt.Sprintf("0.0.0.0:%d", vmDetails.VNC)

// cmd := exec.Command(
// 	q.QemuBin,
// 	"-enable-kvm",
// 	"-m", memory,
// 	"-name", vmDetails.Name,
// 	"-smp", cpu,
// 	"-cpu", "host",
// 	"-drive", driveArg,
// 	"-drive", cloudInitArg,
// 	"-display", "none",
// 	"-vnc", vncArg,
// )
// fmt.Printf("cmd : %v", cmd)
// err = cmd.Start()
// if err != nil {
// 	// logger.LogError(vmDetails.ResourceID, "create-vm", "failed", "failed to create vm", taskId, err) -- same message as orchastrator
// 	return vmDetails, err
// }
// vmDetails.PID = cmd.Process.Pid

// fmt.Println("PID: ", vmDetails.PID)
// vm := *vmDetails
// vm.DiskIndex = 0
// vm.BootDisk = diskPath
// vm.CloudInitFile = cloudInitDisk
// vm.Status = "running"
// vm.DiskType = "boot"
// return &vm, nil
// }

func (q *QEMU) SetupVMArgs(qemuArgs, ifaces []string) []string {
	for i := 0; i < len(ifaces); i++ {
		netdevArg := fmt.Sprintf("tap,id=n%d,ifname=%s,script=no,downscript=no", i, ifaces[i])
		devicArg := fmt.Sprintf("virtio-net-pci,netdev=n%d", i)
		qemuArgs = append(qemuArgs, "-netdev")
		qemuArgs = append(qemuArgs, netdevArg)
		qemuArgs = append(qemuArgs, "-device")
		qemuArgs = append(qemuArgs, devicArg)
	}
	return qemuArgs
}

func (q *QEMU) Rollback(pid, resourceId int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	proc.Signal(syscall.SIGKILL)
	err = os.RemoveAll(filepath.Join(q.DiskStore, strconv.Itoa(resourceId)))
	if err != nil {
		return err
	}
	return nil
}
