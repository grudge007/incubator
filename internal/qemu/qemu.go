package qemu

import (
	"incubator/internal/model"
)

type QEMU struct {
	DiskStore     string
	QemuBin       string
	CloudInitFile string
}

func Manager(vmDetails model.VM) *QEMU {
	return &QEMU{
		DiskStore:     "/var/incubator/disk/",
		QemuBin:       "qemu-system-x86_64",
		CloudInitFile: "/var/incubator/cloudinit/default.yaml",
	}
}
