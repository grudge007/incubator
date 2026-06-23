package qemu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func genDiskPath(diskStore, resourceId string) string {
	return fmt.Sprintf("%s-initial.qcow2", filepath.Join(diskStore, resourceId, resourceId))
}

func genDiskDir(diskStore, resourceId string) string {
	return filepath.Join(diskStore, resourceId)
}

func genDriveArg(diskPath string) string {
	return fmt.Sprintf("file=%s,format=qcow2,if=virtio", diskPath)
}

func createVMDisk(imageFile, diskName, vmName, diskStore string) error {
	d := filepath.Join(diskStore, vmName)
	err := os.MkdirAll(d, 0755)
	if err != nil {
		return err
	}
	fmt.Println(imageFile)
	cmd := exec.Command("qemu-img",
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", imageFile,
		diskName,
	)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func resizeVMDisk(vmDisk string, size int) error {
	diskSize := fmt.Sprintf("%dG", size)
	cmd := exec.Command(
		"qemu-img",
		"resize",
		vmDisk,
		diskSize,
	)
	fmt.Println(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	// fmt.Println(string(ut))
	return nil
}

func (q *QEMU) CreateDataDisk(resourceId string, i, size int) (string, error) {
	sizeArg := fmt.Sprintf("%dG", size)
	disKImage := fmt.Sprintf("%s-data-disk%d.qcow2", filepath.Join(q.DiskStore, resourceId, resourceId), i)
	cmd := exec.Command("qemu-img",
		"create",
		"-f", "qcow2",
		disKImage,
		sizeArg,
	)
	o, err := cmd.CombinedOutput()
	fmt.Println("err: ", string(o))
	if err != nil {
		return "", err
	}
	return disKImage, nil
}

func (q *QEMU) DiskIdGen(i int) string {
	return fmt.Sprintf("virtio%d", i)
}

func (q *QEMU) SetupVmDiskArgs(qemuArgs, disks []string) []string {
	for _, disk := range disks {
		fmt.Printf("\ndisk from arg: %s", disk)
		diskArg := fmt.Sprintf("file=%s,format=qcow2,if=virtio", disk)
		qemuArgs = append(qemuArgs, "-drive")
		qemuArgs = append(qemuArgs, diskArg)
	}
	return qemuArgs
}
