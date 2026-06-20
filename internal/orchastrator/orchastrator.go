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

func (o *Orchastrator) DestroyVMHandler(resourceID string) error {
	var err error

	if o.VM.Status, err = o.Storage.ResourceStatus(resourceID); err != nil {
		return err
	}

	if o.VM.Status != "stopped" {
		return fmt.Errorf("Err: Resource is running")
	}

	if err = o.Qemu.DestroyVM(resourceID); err != nil {
		return err
	}

	if err = o.Storage.DeleteResource("metadata", "resource_id", resourceID); err != nil {
		return err
	}
	return nil

}

func (o *Orchastrator) ShutdownVMHandler(resourceID string) error {

	status, err := o.Storage.ResourceStatus(resourceID)
	if err != nil {
		return err
	}

	if status == "stopped" {
		return fmt.Errorf("Err: Resource is already in stopped state")
	}

	pid, err := o.Storage.FetchPid(resourceID)
	if err != nil {
		return err
	}
	if pid <= 0 {
		return fmt.Errorf("cannot shutdown: invalid or missing PID (%d) for resource %s", pid, resourceID)
	}

	if err = o.Qemu.ShutdownVM(pid); err != nil {
		return err
	}

	if err = o.Storage.UpdateResourceStatusAndPid(resourceID, "stopped", 0); err != nil {
		return fmt.Errorf("resource stopped but failed to update DB state: %w", err)
	}

	return nil
}

func (o *Orchastrator) StartVMHandler(resourceId string) error {
	status, err := o.Storage.ResourceStatus(resourceId)
	if err != nil {
		return err
	}

	if status != "stopped" {
		return fmt.Errorf("Err: Cannot start resource, status is in %s", status)
	}
	vmDetails, err := o.Storage.FetchVmdetails(resourceId)
	if err != nil {
		return err
	}

	pid, err := o.Qemu.StartVM(vmDetails)

	if err != nil {
		return err
	}

	if err = o.Storage.UpdateResourceStatusAndPid(resourceId, "running", pid); err != nil {
		return fmt.Errorf("resource started but failed to update DB state: %w", err)
	}

	return nil
}
