package qemu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func genDiskPath(diskStore, resourceId string) string {
	return fmt.Sprintf("%s.qcow2", filepath.Join(diskStore, resourceId, resourceId))
}
func genDiskDir(diskStore, resourceId string) string {
	return filepath.Join(diskStore, resourceId)
}

func genDriveArg(diskPath string) string {
	return fmt.Sprintf("file=%s,format=qcow2,media=disk", diskPath)
}

func createVMDisk(imageFile, diskName, vmName, diskStore string) error {
	d := filepath.Join(diskStore, vmName)
	err := os.MkdirAll(d, 0440)
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Println(string(out))
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
