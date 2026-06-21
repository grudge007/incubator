package orchastrator

import (
	"context"
	"fmt"
	"incubator/internal/logger"
	"incubator/internal/model"
	"incubator/internal/naming"
	"incubator/internal/network"
	"incubator/internal/qemu"
	"incubator/internal/storage"
	"log"
	"strconv"
)

type Orchastrator struct {
	Qemu    *qemu.QEMU
	Storage *storage.DB
	VM      *model.VM
	Network *network.Network
	Iface   []string
}

func VMManager(db *storage.DB, vmDetails model.VM) *Orchastrator {
	return &Orchastrator{
		Storage: db,
		Qemu:    qemu.Manager(vmDetails),
		VM:      &vmDetails,
		Network: network.InitNetwork(),
	}
}

func (o *Orchastrator) CreateVMHandler(ctx context.Context) error {
	var err error

	o.VM.ResourceID, err = o.Storage.AllocateResourceID()
	if err != nil {
		return err
	}

	imagePath, err := o.Storage.FindImage(o.VM.Image, o.VM.Version)
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to find requested image", err)
		return err
	}

	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully fetched the requested image", imagePath)

	o.VM.VNC, err = o.Storage.AllocateVNCPort()
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to allocate vnc port", err)
		return err
	}

	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully fetched the requested image", strconv.Itoa(o.VM.VNC))

	if o.VM.Name == "auto" {
		o.VM.Name = naming.GenerateVMName()
		logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully allocated resource name", o.VM.Name)
	}

	vmResp, qemuArgs, err := o.Qemu.PrepareVMCreation(ctx, o.VM, imagePath)
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to create vm", err)
		return err
	}
	fmt.Printf("Resp : %v", vmResp)
	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully crraeted and started vm", vmResp.Name)

	_, err = o.Storage.InsertInitialVmMeta(vmResp)
	if err != nil {
		return err
	}

	// fmt.Println("meta id: ", metaId)
	fmt.Println("", len(o.Iface))

	var ifaces []string
	for i := 0; i < len(o.Iface); i++ {
		ifaceName := o.Network.GenerateTapDevName(i, vmResp.ResourceID)
		ifaces = append(ifaces, ifaceName)
		var bridge string

		if o.Iface[i] == "default" {
			bridge = o.Network.DefaultBridge
		} else {
			bridge = o.Iface[i]
		}
		ok, link, err := o.Network.CheckBridgeExist(bridge)
		if err != nil {
			return err
		}
		if !ok {
			return err
		}

		ok, err = o.Network.CheckTapBridgePortExist(ifaceName)
		if err != nil {
			return err
		}

		if !ok {
			o.Network.CreateTapBridgePort(link, ifaceName)
		}
		bridgeId, err := o.Storage.FetchBridgeId(o.Iface[i])
		fmt.Println("bridge_id", bridgeId)

		if err != nil {
			return err
		}

		err = o.Storage.InserNetworkIface(vmResp.ResourceID, bridgeId, ifaceName)
		if err != nil {
			return err
		}

	}

	ifaces, err = o.Storage.FetchInterfaces(vmResp.ResourceID)
	if err != nil {
		return err
	}

	qemuArgs = o.Qemu.SetupVMArgs(qemuArgs, ifaces)
	fmt.Println(qemuArgs)
	pid, err := o.Qemu.StartVM(ctx, qemuArgs)
	if err != nil {
		return err
	}

	err = o.Storage.UpdateResourceStatusAndPid(strconv.Itoa(vmResp.ResourceID), "running", pid)
	if err != nil {
		return err
	}

	fmt.Printf("pid: %d\n qemuArgs: %v\n", pid, qemuArgs)

	// if err != nil {
	// 	logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to insert metadata to DB", err)

	// 	err = o.Qemu.Rollback(o.VM.PID, o.VM.ResourceID)
	// 	if err != nil {
	// 		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to rollback", err)
	// 	} else {
	// 		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "successfully rolled back resource state", nil)
	// 	}

	// 	return err
	// }

	// logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully updated state", vmResp.Name)

	return nil
}

func (o *Orchastrator) DestroyVMHandler(ctx context.Context, resourceID string) error {
	var err error
	resId, _ := strconv.Atoi(resourceID)

	if o.VM.Status, err = o.Storage.ResourceStatus(resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to fetch resource status", err)
		return err
	}

	if o.VM.Status != "stopped" {
		err = fmt.Errorf("Err: Resource is running")
		logger.LogError(ctx, resId, "destroy-vm", "cannot destroy running resource", err)
		return err
	}

	if err = o.Qemu.DestroyVM(ctx, resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to destroy vm via qemu", err)
		return err
	}
	ifaces, err := o.Storage.FetchInterfaces(resId)
	if err != nil {
		return err
	}

	for _, iface := range ifaces {
		err = o.Network.DeleteTapFromBridgeAndSystem(iface)
		if err != nil {
			return err
		}
	}

	if err = o.Storage.DeleteResource("metadata", "resource_id", resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to delete resource from database", err)
		return err
	}

	logger.LogSuccess(ctx, resId, "destroy-vm", "successfully destroyed resource", resourceID)
	return nil

}

func (o *Orchastrator) ShutdownVMHandler(ctx context.Context, resourceID string) error {
	resId, _ := strconv.Atoi(resourceID)

	status, err := o.Storage.ResourceStatus(resourceID)
	if err != nil {
		logger.LogError(ctx, resId, "stop-vm", "failed to fetch resource status", err)
		return err
	}

	if status == "stopped" {
		err = fmt.Errorf("Err: Resource is already in stopped state")
		logger.LogError(ctx, resId, "stop-vm", "cannot stop already stopped resource", err)
		return err
	}

	pid, err := o.Storage.FetchPid(resourceID)
	if err != nil {
		logger.LogError(ctx, resId, "stop-vm", "failed to fetch pid", err)
		return err
	}
	if pid <= 0 {
		err = fmt.Errorf("cannot shutdown: invalid or missing PID (%d) for resource %s", pid, resourceID)
		logger.LogError(ctx, resId, "stop-vm", "invalid pid", err)
		return err
	}

	if err = o.Qemu.ShutdownVM(ctx, pid); err != nil {
		logger.LogError(ctx, resId, "stop-vm", "failed to shutdown vm via qemu", err)
		return err
	}

	if err = o.Storage.UpdateResourceStatusAndPid(resourceID, "stopped", 0); err != nil {
		err = fmt.Errorf("resource stopped but failed to update DB state: %w", err)
		logger.LogError(ctx, resId, "stop-vm", "failed to update database state", err)
		return err
	}

	logger.LogSuccess(ctx, resId, "stop-vm", "successfully stopped resource", resourceID)
	return nil
}

func (o *Orchastrator) StartVMHandler(ctx context.Context, resourceId string) error {
	resId, _ := strconv.Atoi(resourceId)

	status, err := o.Storage.ResourceStatus(resourceId)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to fetch resource status", err)
		return err
	}

	if status != "stopped" {
		err = fmt.Errorf("Err: Cannot start resource, status is in %s", status)
		logger.LogError(ctx, resId, "start-vm", "resource is not stopped", err)
		return err
	}
	// _, err := o.Storage.FetchVmdetails(resourceId)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to fetch vm details", err)
		return err
	}

	pid, err := o.Qemu.StartVM(ctx, o.Iface)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to start vm via qemu", err)
		return err
	}

	if err = o.Storage.UpdateResourceStatusAndPid(resourceId, "running", pid); err != nil {
		err = fmt.Errorf("resource started but failed to update DB state: %w", err)
		logger.LogError(ctx, resId, "start-vm", "failed to update database state", err)
		return err
	}

	logger.LogSuccess(ctx, resId, "start-vm", "successfully started resource", resourceId)
	return nil
}

func (o *Orchastrator) VmStatusHandler(ctx context.Context, resourceId string) error {
	var status string

	pid, err := o.Storage.FetchPid(resourceId)
	resId, _ := strconv.Atoi(resourceId)
	if err != nil {
		logger.LogError(ctx, resId, "vm-status", "failed to fetch pid", err)
		return err
	}
	status = "running"

	running := o.Qemu.IsProcessRunning(pid)
	// if pid is inactive and non-zero, it means vm is dead but db is not updated

	if !running {
		status = "stopped"
		pid = 0

		err = o.Storage.UpdateResourceStatusAndPid(resourceId, "stopped", pid)
		if err != nil {
			logger.LogError(ctx, resId, "vm-status", "failed to update vm state", err)
			return err
		}
		logger.LogSuccess(ctx, resId, "vm-status", "successfully updated vm state", resourceId)

	}

	fmt.Println("========================================")
	fmt.Println("VM STATUS")
	fmt.Println("========================================")

	fmt.Printf("%-15s: %d\n", "Resource ID", resId)
	fmt.Printf("%-15s: %s\n", "Status", status)
	fmt.Printf("%-15s: %d\n", "PID", pid)

	fmt.Println("========================================")
	logger.LogSuccess(ctx, resId, "vm-status", "successfully listed vm status", resourceId)
	return nil
}

func (o *Orchastrator) ListResources(ctx context.Context, resourceId string) error {
	var vms []model.VM
	var err error
	var show_empty bool
	if resourceId == "" {
		vms, err = o.Storage.ListAllResource()
		if err != nil {
			logger.LogError(ctx, 0, "list-vm", "failed to list all resources", err)
			log.Fatalf("Error getting VM list: %v", err)
		}

	} else {
		vms, err = o.Storage.ListSingleResource(resourceId)
		if err != nil {
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
