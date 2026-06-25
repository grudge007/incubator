package qemu

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func (q *QEMU) CreateCloudInitIso(cloudInitFile, diskDir string) (string, error) {
	cloudInitDisk := filepath.Join(diskDir, "cloud-init-seed.img")
	cmd := exec.Command(
		"cloud-localds",
		cloudInitDisk,
		cloudInitFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	fmt.Println(string(out))
	return cloudInitDisk, nil
}
