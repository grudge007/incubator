package orchastrator

import (
	"fmt"
	"incubator/internal/model"
	"incubator/internal/naming"
	"incubator/internal/qemu"
	"incubator/internal/storage"
	"log"
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

func (o *Orchastrator) ListResources(resourceId string) error {
	var vms []model.VM
	var err error
	var show_empty bool
	if resourceId == "" {
		vms, err = o.Storage.ListAllResource()
		if err != nil {
			log.Fatalf("Error getting VM list: %v", err)
		}

	} else {
		vms, err = o.Storage.ListSingleResource(resourceId)
		if err != nil {
			// return fmt.Errorf("Error getting VM list: %v", err)
			show_empty = true
		}

	}

	fmt.Println("=================================================================================================================================")
	fmt.Printf("%-12s %-25s %-6s %-10s %-8s %-12s %-15s %-10s %-10s\n",
		"RES ID", "VM NAME", "CPUS", "MEMORY", "VNC", "PID", "OS IMAGE", "STATUS", "DISK")
	fmt.Println("=================================================================================================================================")

	if show_empty || len(vms) == 0 {
		fmt.Println("                                                    No resources found.                                                          ")
		fmt.Println("=================================================================================================================================")
		return nil
	}

	for _, vm := range vms {
		// Matching data types:
		// %-12d (Int ResID), %-18s (String Name), %-6d (Int CPUs), %-10s (Formatted Memory),
		// %-8d (Int VNC), %-12d (Int PID), %-15s (String Image), %-10s (String Status), %s (Disk Size)
		fmt.Printf("%-12d %-25s %-6d %-d MB   %-8d %-12d %-15s %-10s %d GB\n",
			vm.ResourceID,
			vm.Name,
			vm.CPUs,
			vm.MemoryMB,
			vm.VNC,
			vm.PID,
			vm.Image,
			vm.Status,
			vm.DiskSize,
		)
	}
	fmt.Println("=================================================================================================================================")
	return nil
}
