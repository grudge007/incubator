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

func (q *QEMU) PrepareVMCreation(ctx context.Context, vmDetails *model.VM, imagePath string) (*model.VM, error) {

	resourceId := strconv.Itoa(vmDetails.ResourceID)
	diskDir := genDiskDir(q.DiskStore, resourceId)
	fmt.Println("%#W", diskDir)
	diskPath := genDiskPath(q.DiskStore, resourceId)

	err := createVMDisk(imagePath, diskPath, resourceId, q.DiskStore)
	if err != nil {
		logger.LogError(ctx, vmDetails.ResourceID, "create-vm", "failed to create vm disk", err)
		return vmDetails, err
	}
	logger.LogSuccess(ctx, vmDetails.ResourceID, "create-vm", "succesfully created vm disk", imagePath)

	err = resizeVMDisk(diskPath, vmDetails.DiskSize)
	if err != nil {
		logger.LogError(ctx, vmDetails.ResourceID, "create-vm", "failed to resize vm disk", err)
		return vmDetails, err
	}

	logger.LogSuccess(ctx, vmDetails.ResourceID, "create-vm", "succesfully resized vm disk", strconv.Itoa(vmDetails.DiskSize))

	cloudInitDisk, err := createCloudInitIso(q.CloudInitFile, diskDir)
	if err != nil {
		logger.LogError(ctx, vmDetails.ResourceID, "create-vm", "failed to generate cloud init disk", err)
		return vmDetails, err
	}

	vm := *vmDetails
	vm.DiskIndex = 0
	vm.BootDisk = diskPath
	vm.CloudInitFile = cloudInitDisk
	vm.DiskType = "boot"
	return &vm, nil
}

func (q *QEMU) SetupVMIfaceArgs(qemuArgs, ifaces []string) []string {
	for i := range len(ifaces) {
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
	if pid > 0 {
		proc, err := os.FindProcess(pid)
		if err != nil {
			// log here, no need to stop...
		}
		proc.Signal(syscall.SIGKILL)

	}
	err := os.RemoveAll(filepath.Join(q.DiskStore, strconv.Itoa(resourceId)))
	if err != nil {
		return err
	}

	return nil
}
