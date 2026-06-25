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
	"strconv"
)

type Orchastrator struct {
	Qemu          *qemu.QEMU
	Storage       *storage.DB
	VM            *model.VM
	DefaultBridge string
	Network       network.BridgeManager
	PortMgr       network.PortManager
	Interface     *model.Interface
	Bridges       []string
	Disk          []int
}

func VMManager(db *storage.DB, vmDetails model.VM, netMgr network.BridgeManager) *Orchastrator {
	return &Orchastrator{
		Storage:       db,
		Qemu:          qemu.Manager(vmDetails),
		VM:            &vmDetails,
		Network:       netMgr,
		DefaultBridge: "inbr0",
		Interface:     &model.Interface{},
		PortMgr:       network.NewTapDevManager(),
	}
}

func (o *Orchastrator) CreateVMHandler(ctx context.Context) error {
	var err error
	var pid int
	var ifaces []string
	var bridgeType string

	o.VM.ResourceID, err = o.Storage.AllocateResourceID()
	if err != nil {
		logger.LogError(ctx, 0, "create-vm", "failed to allocate resource ID", err)
		return fmt.Errorf("failed to allocate resource ID: %w", err)
	}

	imagePath, err := o.Storage.FindImage(o.VM.Image, o.VM.Version)
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to find requested image", err)
		return fmt.Errorf("failed to find requested image: %w", err)
	}

	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully fetched the requested image", imagePath)

	o.VM.VNC, err = o.Storage.AllocateVNCPort()
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to allocate vnc port", err)
		return fmt.Errorf("failed to allocate vnc port: %w", err)
	}

	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "successfully allocated vnc port", strconv.Itoa(o.VM.VNC))

	if o.VM.Name == "auto" {
		o.VM.Name = naming.GenerateVMName()
		logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "succesfully allocated resource name", o.VM.Name)
	}

	vmResp, err := o.Qemu.PrepareVMCreation(ctx, o.VM, imagePath)
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to create vm", err)
		return fmt.Errorf("failed to create vm: %w", err)
	}
	fmt.Printf("Resp : %v", vmResp)
	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "successfully prepared VM disks", vmResp.Name)

	_, err = o.Storage.InsertInitialVmMeta(vmResp)
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to insert initial VM metadata", err)
		o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
		return fmt.Errorf("failed to insert initial VM metadata: %w", err)
	}

	// fmt.Println("meta id: ", metaId)
	fmt.Println("\n", len(o.Bridges))
	fmt.Println("o.Bridges", o.Bridges)
	for i := range len(o.Bridges) {
		var bridge string
		if o.Bridges[i] == "default" {
			bridge = o.DefaultBridge
		} else {
			bridge = o.Bridges[i]
		}
		bridgeType, err = o.Storage.FetchBridgeType(bridge)
		if err != nil {
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return err
		}

		netMgr, err := network.NewBridgeManager(bridgeType)
		if err != nil {
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return err
		}
		ifaceName := o.PortMgr.GenerateTapDevName(i, vmResp.ResourceID)
		ifaces = append(ifaces, ifaceName)

		fmt.Println("ifaces: ", ifaces)

		ok, err := netMgr.CheckBridgeExist(bridge)
		if err != nil {
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to check bridge exist", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return fmt.Errorf("failed to check bridge exist: %w", err)
		}
		if !ok {
			err = fmt.Errorf("bridge %s does not exist", bridge)
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "bridge not found", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return err
		}

		ok, err = o.PortMgr.CheckTapBridgePortExist(ifaceName)
		if err != nil {
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to check tap port exist", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return fmt.Errorf("failed to check tap port exist: %w", err)
		}

		if !ok {
			err = o.PortMgr.CreateTapPort(ifaceName)
			if err != nil {
				logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to create tap bridge port", err)
				o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
				return fmt.Errorf("failed to create tap bridge port: %w", err)
			}
			err = netMgr.AttachTapDevToBridge(bridge, ifaceName)
			if err != nil {
				logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to create tap bridge port", err)
				o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
				return fmt.Errorf("failed to create tap bridge port: %w", err)
			}
		}
		fmt.Printf("\nbridge: %s\n", bridge)
		bridgeId, err := o.Storage.FetchBridgeId(bridge)
		fmt.Println("bridge_id", bridgeId)

		if err != nil {
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to fetch bridge id", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return fmt.Errorf("failed to fetch bridge id: %w", err)
		}

		err = o.Storage.InserNetworkIface(vmResp.ResourceID, bridgeId, ifaceName)
		if err != nil {
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to insert network interface into db", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return fmt.Errorf("failed to insert network interface into db: %w", err)
		}

	}

	var diskImages []string

	for i := 0; i < len(o.Disk); i++ {
		diskImage, err := o.Qemu.CreateDataDisk(strconv.Itoa(o.VM.ResourceID), i, o.Disk[i])
		fmt.Println("i: ", i)
		if err != nil {
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to create data disk", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return fmt.Errorf("failed to create data disk: %w", err)
		}

		diskId := o.Qemu.DiskIdGen(i)

		err = o.Storage.InsertDataDisk(vmResp.ResourceID, diskImage, diskId, false)
		if err != nil {
			logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to insert data disk details into db", err)
			o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
			return fmt.Errorf("failed to insert data disk details into db: %w", err)
		}
		diskImages = append(diskImages, diskImage)
		fmt.Printf("diskImages: %v", diskImages)

	}

	qemuArgs := o.Qemu.PrepareQemuArgs(*vmResp)

	qemuArgs = o.Qemu.SetupVMIfaceArgs(qemuArgs, ifaces)

	qemuArgs = o.Qemu.SetupVmDiskArgs(qemuArgs, diskImages)

	pid, err = o.Qemu.StartVM(ctx, qemuArgs)
	if err != nil {
		fmt.Println("\nfuckedup")
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to start vm via qemu", err)
		o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
		return fmt.Errorf("failed to start vm via qemu: %w", err)
	}

	err = o.Storage.UpdateResourceStatusAndPid(strconv.Itoa(vmResp.ResourceID), "running", pid)
	if err != nil {
		logger.LogError(ctx, o.VM.ResourceID, "create-vm", "failed to update status to running", err)
		o.RollbackHandler(ctx, pid, o.VM.ResourceID, ifaces, bridgeType)
		return fmt.Errorf("failed to update status to running: %w", err)
	}

	fmt.Printf("pid: %d\n qemuArgs: %v\n", pid, qemuArgs)
	logger.LogSuccess(ctx, o.VM.ResourceID, "create-vm", "successfully created and started vm", vmResp.Name)
	return nil
}

func (o *Orchastrator) DestroyVMHandler(ctx context.Context, resourceID string) error {
	var err error
	resId, _ := strconv.Atoi(resourceID)

	if o.VM.Status, err = o.Storage.ResourceStatus(resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to fetch resource status", err)
		return fmt.Errorf("failed to fetch resource status: %w", err)
	}

	if o.VM.Status != "stopped" {
		err = fmt.Errorf("Err: Resource is running")
		logger.LogError(ctx, resId, "destroy-vm", "cannot destroy running resource", err)
		return err
	}

	if err = o.Qemu.DestroyVM(ctx, resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to destroy vm via qemu", err)
		return fmt.Errorf("failed to destroy vm via qemu: %w", err)
	}
	var ifaceData []model.ResourceIfaceInfo
	ifaceData, err = o.Storage.FetchIfaceInfoWithResId(resId)

	// ifaces, err := o.Storage.FetchInterfaces(resId)
	// if err != nil {
	// 	logger.LogError(ctx, resId, "destroy-vm", "failed to fetch interfaces", err)
	// 	return fmt.Errorf("failed to fetch interfaces: %w", err)
	// }
	for i, ifaceInfo := range ifaceData {
		netMgr, err := network.NewBridgeManager(ifaceInfo.BridgeType)
		if err != nil {
			// logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
			return fmt.Errorf("failed to delete tap interface %s: %w", err)
		}
		err = netMgr.DeleteTapFromBridge(ifaceInfo.IfaceName)
		if err != nil {
			// logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
			return fmt.Errorf("failed to delete tap interface %s: %w", err)
		}
		err = o.PortMgr.DeleteTapFromSystem(ifaceInfo.IfaceName)
		if err != nil {
			// logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
			return fmt.Errorf("failed to delete tap interface %s: %w", err)
		}
		fmt.Println(i)
	}

	// for _, iface := range ifaces {
	// 	o.Storage.Fetch
	// 	err = o.PortMgr.DeleteTapFromBridge(iface)
	// 	if err != nil {
	// 		logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
	// 		return fmt.Errorf("failed to delete tap interface %s: %w", iface, err)
	// 	}
	// }

	if err = o.Storage.DeleteResource("networks", "resource_id", resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to delete networks from database", err)
		return fmt.Errorf("failed to delete networks from database: %w", err)
	}

	if err = o.Storage.DeleteResource("data_disks", "resource_id", resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to delete data disks from database", err)
		return fmt.Errorf("failed to delete data disks from database: %w", err)
	}

	if err = o.Storage.DeleteResource("metadata", "resource_id", resourceID); err != nil {
		logger.LogError(ctx, resId, "destroy-vm", "failed to delete resource from database", err)
		return fmt.Errorf("failed to delete resource from database: %w", err)
	}

	logger.LogSuccess(ctx, resId, "destroy-vm", "successfully destroyed resource", resourceID)
	return nil

}

func (o *Orchastrator) ShutdownVMHandler(ctx context.Context, resourceID string) error {
	resId, _ := strconv.Atoi(resourceID)

	status, err := o.Storage.ResourceStatus(resourceID)
	if err != nil {
		logger.LogError(ctx, resId, "stop-vm", "failed to fetch resource status", err)
		return fmt.Errorf("failed to fetch resource status: %w", err)
	}

	if status == "stopped" {
		err = fmt.Errorf("Err: Resource is already in stopped state")
		logger.LogError(ctx, resId, "stop-vm", "cannot stop already stopped resource", err)
		return err
	}

	pid, err := o.Storage.FetchPid(resourceID)
	if err != nil {
		logger.LogError(ctx, resId, "stop-vm", "failed to fetch pid", err)
		return fmt.Errorf("failed to fetch pid: %w", err)
	}
	if pid <= 0 {
		err = fmt.Errorf("cannot shutdown: invalid or missing PID (%d) for resource %s", pid, resourceID)
		logger.LogError(ctx, resId, "stop-vm", "invalid pid", err)
		return err
	}

	if err = o.Qemu.ShutdownVM(ctx, pid); err != nil {
		logger.LogError(ctx, resId, "stop-vm", "failed to shutdown vm via qemu", err)
		return fmt.Errorf("failed to shutdown vm via qemu: %w", err)
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
		return fmt.Errorf("failed to fetch resource status: %w", err)
	}

	if status != "stopped" {
		err = fmt.Errorf("Err: Cannot start resource, status is in %s", status)
		logger.LogError(ctx, resId, "start-vm", "resource is not stopped", err)
		return err
	}
	vmDetails, err := o.Storage.FetchVmdetails(resourceId)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to fetch vm details", err)
		return fmt.Errorf("failed to fetch vm details: %w", err)
	}

	qemuArgs := o.Qemu.PrepareQemuArgs(vmDetails)
	fmt.Printf("\nQEMU ARGS: %v\n", qemuArgs)
	ifaces, err := o.Storage.FetchInterfaces(vmDetails.ResourceID)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to fetch interfaces", err)
		return fmt.Errorf("failed to fetch interfaces: %w", err)
	}

	diskImages, err := o.Storage.FetchDataDisks(resourceId)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to fetch data disks", err)
		return fmt.Errorf("failed to fetch data disks: %w", err)
	}

	qemuArgs = o.Qemu.SetupVMIfaceArgs(qemuArgs, ifaces)
	qemuArgs = o.Qemu.SetupVmDiskArgs(qemuArgs, diskImages)

	pid, err := o.Qemu.StartVM(ctx, qemuArgs)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to start vm via qemu", err)
		return fmt.Errorf("failed to start vm via qemu: %w", err)
	}
	fmt.Printf("Vm Details: %v \n", vmDetails)
	fmt.Printf("Passed ID : %d\n, From DB: %d\n", resId, vmDetails.ResourceID)
	err = o.Storage.UpdateResourceStatusAndPid(strconv.Itoa(resId), "running", pid)
	if err != nil {
		logger.LogError(ctx, resId, "start-vm", "failed to update database state", err)
		return fmt.Errorf("failed to update database state: %w", err)
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
		return fmt.Errorf("failed to fetch pid: %w", err)
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
			return fmt.Errorf("failed to update vm state: %w", err)
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
	if resourceId == "" {
		vms, err = o.Storage.ListAllResource()
		if err != nil {
			logger.LogError(ctx, 0, "list-vm", "failed to list all resources", err)
			return fmt.Errorf("failed to list all resources: %w", err)
		}

	} else {
		vms, err = o.Storage.ListSingleResource(resourceId)
		if err != nil {
			resId, _ := strconv.Atoi(resourceId)
			logger.LogError(ctx, resId, "list-vm", "failed to list single resource", err)
			return fmt.Errorf("failed to list resource %s: %w", resourceId, err)
		}
	}

	fmt.Println("=================================================================================================================================")
	fmt.Printf("%-12s %-25s %-6s %-10s %-8s %-12s %-15s %-10s %-10s\n",
		"RES ID", "VM NAME", "CPUS", "MEMORY", "VNC", "PID", "OS IMAGE", "STATUS", "DISK")
	fmt.Println("=================================================================================================================================")

	if len(vms) == 0 {
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

func (o *Orchastrator) RollbackHandler(ctx context.Context, pid, resourceId int, ifaces []string, bridgeType string) {
	o.Qemu.Rollback(pid, resourceId)
	var ifaceData []model.ResourceIfaceInfo
	ifaceData, err := o.Storage.FetchIfaceInfoWithResId(resourceId)

	// ifaces, err := o.Storage.FetchInterfaces(resId)
	// if err != nil {
	// 	logger.LogError(ctx, resId, "destroy-vm", "failed to fetch interfaces", err)
	// 	return fmt.Errorf("failed to fetch interfaces: %w", err)
	// }
	for _, ifaceInfo := range ifaceData {
		netMgr, err := network.NewBridgeManager(ifaceInfo.BridgeType)
		if err != nil {
			// logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
			// return fmt.Errorf("failed to delete tap interface %s: %w", err)
		}
		err = netMgr.DeleteTapFromBridge(ifaceInfo.IfaceName)
		if err != nil {
			// logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
			// return fmt.Errorf("failed to delete tap interface %s: %w", err)
		}
		err = o.PortMgr.DeleteTapFromSystem(ifaceInfo.IfaceName)
		if err != nil {
			// logger.LogError(ctx, resId, "destroy-vm", fmt.Sprintf("failed to delete tap interface %s", iface), err)
			// return fmt.Errorf("failed to delete tap interface %s: %w", err)
		}
	}

	err = o.Storage.DeleteResource("networks", "resource_id", strconv.Itoa(resourceId))
	if err != nil {
		logger.LogError(ctx, resourceId, "create-vm", "failed to delete networks from db during rollback", err)
	}
	err = o.Storage.DeleteResource("data_disks", "resource_id", strconv.Itoa(resourceId))
	if err != nil {
		logger.LogError(ctx, resourceId, "create-vm", "failed to delete data disks from db during rollback", err)
	}
	err = o.Storage.DeleteResource("metadata", "resource_id", strconv.Itoa(resourceId))
	if err != nil {
		logger.LogError(ctx, resourceId, "create-vm", "failed to delete from db during rollback", err)
		return
	}
	logger.LogError(ctx, resourceId, "create-vm", "successfully rolled back resource", nil)
}

func (o *Orchastrator) CreateBridgeHandler(ctx context.Context) error {

	ok, err := o.Network.CheckBridgeExist(o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "create-bridge", "failed to check if bridge exists", err)
		return fmt.Errorf("failed to check if bridge exists: %w", err)
	}

	if ok {
		err = fmt.Errorf("bridge %s already exists", o.Interface.Name)
		logger.LogError(ctx, 0, "create-bridge", "bridge already exists", err)
		return err
	}

	err = o.Network.CreateBridge(o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "create-bridge", "failed to create linux bridge", err)
		return fmt.Errorf("failed to create linux bridge: %w", err)
	}

	err = o.Storage.InsertBridgeDetails(o.Interface.Name, o.Interface.Type)
	if err != nil {
		logger.LogError(ctx, 0, "create-bridge", "failed to insert bridge details into db", err)
		cleanupErr := o.Network.DeleteBridgeByName(o.Interface.Name)
		if cleanupErr != nil {
			logger.LogError(ctx, 0, "create-bridge", "failed to delete bridge during cleanup", cleanupErr)
		}
		return fmt.Errorf("failed to insert bridge details into db: %w", err)
	}
	logger.LogSuccess(ctx, 0, "create-bridge", "successfully created bridge", o.Interface.Name)
	// case "ovs":

	// default:
	// 	err := fmt.Errorf("invalid bridge type: %s", o.Interface.Type)
	// 	logger.LogError(ctx, 0, "create-bridge", "invalid network type", err)
	// 	return err
	// }
	return nil
}

func (o *Orchastrator) DeleteBridgeHandler(ctx context.Context) error {
	bridgeType, err := o.Storage.FetchBridgeType(o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "delete-bridge", "failed to fetch bridge type from db", err)
		return fmt.Errorf("failed to fetch bridge type from db: %w", err)
	}
	netMgr, err := network.NewBridgeManager(bridgeType)

	inuse, err := o.Storage.VerifyIsBridgeIdle(o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "delete-bridge", "failed to fetch bridge availability", err)
		return fmt.Errorf("failed to fetch bridge availability: %w", err)
	}
	if inuse {
		err = fmt.Errorf("bridge %s is in use", o.Interface.Name)
		logger.LogError(ctx, 0, "delete-bridge", "bridge is in use", err)
		return err
	}

	ok, err := netMgr.CheckBridgeExist(o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "delete-bridge", "failed to check if bridge exists", err)
		return fmt.Errorf("failed to check if bridge exists: %w", err)
	}

	if !ok {
		err = fmt.Errorf("bridge %s does not exist", o.Interface.Name)
		logger.LogError(ctx, 0, "delete-bridge", "bridge not found", err)
		return err
	}

	err = netMgr.DeleteBridgeByName(o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "delete-bridge", "failed to delete bridge", err)
		return fmt.Errorf("failed to delete bridge: %w", err)
	}

	err = o.Storage.DeleteResource("network_bridges", "name", o.Interface.Name)
	if err != nil {
		logger.LogError(ctx, 0, "delete-bridge", "failed to delete bridge from db", err)
		return fmt.Errorf("failed to delete bridge from db: %w", err)
	}
	logger.LogSuccess(ctx, 0, "delete-bridge", "successfully deleted bridge", o.Interface.Name)

	return nil
}

func (o *Orchastrator) ListBridges() error {
	var bridgeDetails []model.ResourceIfaceInfo
	bridgeDetails, err := o.Storage.FetchBridgeDetails()
	if err != nil {
		return err
	}

	fmt.Println(bridgeDetails)
	return nil
}

// func (o *Orchastrator) RegenerateCloudInit(resourceId int) error {

// }
