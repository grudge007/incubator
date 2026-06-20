package orchastrator

import (
	"fmt"
	"incubator/internal/model"
	"incubator/internal/naming"
	"incubator/internal/qemu"
	"incubator/internal/storage"
)

type Orchastrator struct {
	Qemu    *qemu.QEMU
	Storage *storage.DB
	VM      *model.VM
}

func VMManager(db *storage.DB, vmDetails *model.VM) *Orchastrator {
	return &Orchastrator{
		Storage: db,
		Qemu:    qemu.Manager(*vmDetails),
		VM:      vmDetails,
	}
}

func (o *Orchastrator) CreateVMHandler() error {
	imagePath, err := o.Storage.FindImage(o.VM.Image, o.VM.Version)
	if err != nil {
		return err
	}
	o.VM.VNC, err = o.Storage.AllocateVNCPort()
	if err != nil {
		return err
	}
	o.VM.ResourceID, err = o.Storage.AllocateResourceID()
	if err != nil {
		return err
	}
	o.VM.Name = naming.GenerateVMName()

	fmt.Printf("\nVNC FROM ORCA: %d\n", o.VM.VNC)
	fmt.Printf("\nVMID FROM ORCA: %d\n", o.VM.ResourceID)

	vmResp, err := o.Qemu.CreateVM(o.VM, imagePath)
	if err != nil {
		return err
	}

	err = o.Storage.InsertVmMeta(vmResp)
	if err != nil {
		return err
	}
	fmt.Printf("Resp: %v", vmResp)
	return nil
}
