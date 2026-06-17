package qemu

import (
	"encoding/json"
	"fmt"
	"incubator/internal/images"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const ImagePath = "/var/incubator/images/"
const diskStore = "/var/incubator/disk/"
const metaData = "/var/incubator/.meta/meta.json"

type VM struct {
	PID           int    `json:"pid"`
	CPUs          int    `json:"cpu"`
	MemoryMB      int    `json:"memory_mb"`
	SSHPort       int    `json:"ssh_port"`
	DiskSize      int    `json:"disk_size"`
	Name          string `json:"resource_name"`
	DiskPath      string `json:"disk_path"`
	CloudInitFile string `json:"cloud_init"`
	Image         string `json:"os_image"`
	Status        string `json:"status"`
}

type VmListing struct {
	PID    int
	Name   string
	Image  string
	Status string
}
type META struct {
	ResourceName string `json:"resource_name"`
	MetaData     VM
}

type CloudInit struct {
	Users []User `yaml:"users"`
}
type User struct {
	Name       string   `yaml:"name"`
	Sudo       string   `yaml:"sudo"`
	LockPasswd bool     `yaml:"lock_passwd"`
	Shell      string   `yaml:"shell"`
	SshKey     []string `yaml:"ssh_authorized_keys"`
}

type Action struct {
	Action string
}

type VMManager struct {
	ImagePath string
	// Image     string
}

func NewVMManager() *VMManager {
	return &VMManager{
		ImagePath: ImagePath,
		// Image:     image,
	}
}

//
// Create VM
//

func (vm *VMManager) CreateVM(vmDetails VM, availableImages images.AvailableImages) error {
	memStr := strconv.Itoa(vmDetails.MemoryMB)
	cpuStr := strconv.Itoa(vmDetails.CPUs)
	sshFwd := fmt.Sprintf("tcp::%d-:22", vmDetails.SSHPort)
	driveArg := fmt.Sprintf("file=%s,format=qcow2,media=disk", vmDetails.DiskPath)
	diskName := vmDetails.DiskPath

	imageFile, err := findImageFile(vmDetails.Image, availableImages)
	if err != nil {
		return err
	}
	err = vm.createVMDisk(imageFile, diskName, vmDetails.Name)
	if err != nil {
		return err
	}

	err = vm.resizeVMDisk(diskName, vmDetails.DiskSize)
	if err != nil {
		return err
	}
	// fmt.Println(vmDetails.CloudInitFile)

	if err := createCloudInitIso(vmDetails.CloudInitFile); err != nil {
		return err
	}

	fmt.Println(vmDetails)

	cmd := exec.Command(
		"qemu-system-x86_64",
		"-enable-kvm",
		"-m", memStr,
		"-name", vmDetails.Name,
		"-smp", cpuStr,
		"-cpu", "host",
		"-drive", driveArg,
		"-drive", "file=seed.img,format=raw,media=cdrom",
		"-netdev", "user,id=n1,hostfwd="+sshFwd,
		"-device", "virtio-net-pci,netdev=n1",
		// "-nographic",
		"-display", "none", // Disables the local QEMU window on the host
		"-vnc", "0.0.0.0:1",
	)
	fmt.Println(cmd)
	err = cmd.Start()
	if err != nil {
		return err
	}
	vmDetails.PID = cmd.Process.Pid
	fmt.Println("PID: ", vmDetails.PID)
	StoreMetadata(vmDetails)
	return nil

}

func (vm *VMManager) createVMDisk(imageFile, diskName, name string) error {
	d := filepath.Join(diskStore, name)
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

func (vm *VMManager) resizeVMDisk(vmDisk string, size int) error {
	diskSize := fmt.Sprintf("%dG", size)
	cmd := exec.Command(
		"qemu-img",
		"resize",
		vmDisk,
		diskSize,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func createCloudInitIso(cloudInitFile string) error {
	cmd := exec.Command(
		"cloud-localds",
		"seed.img",
		cloudInitFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func findImageFile(image string, availableImages images.AvailableImages) (string, error) {
	// strings.Contains(image, "")
	versions, ok := availableImages.Images[image]
	if !ok {
		return "", fmt.Errorf("unknown image: %s", image)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions configured for %s", image)
	}

	return versions[0].Path, nil

}

func StoreMetadata(vmDetails VM) error {
	var meta []META
	var vmMeta META
	data, err := os.ReadFile(metaData)
	if err == nil && len(data) > 0 {
		if err = json.Unmarshal(data, &meta); err != nil {
			return fmt.Errorf("failed to parse existing JSON: %w", err)
		}
	}
	vmMeta = META{
		ResourceName: vmDetails.Name,
		MetaData:     vmDetails,
	}
	meta = append(meta, vmMeta)
	updateData, err := json.MarshalIndent(meta, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	err = os.WriteFile(metaData, updateData, 0666)
	if err != nil {
		return fmt.Errorf("failed to write data to %s: %w", metaData, err)
	}
	return nil
}

//
//  ShutDown
//

func (vm *VMManager) ShutdownVM(resourceName string) error {
	pid, err := findResourcePID(resourceName)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// proc.Signal(syscall.SIGKILL)
	proc.Signal(syscall.SIGTERM)
	for range 5 {
		if !isProcessRunning(pid) {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	if isProcessRunning(pid) {
		proc.Signal(syscall.SIGKILL)
	}
	updateMeta(resourceName)
	return nil
}

func findResourcePID(resourceName string) (int, error) {
	var meta []META
	file, err := os.ReadFile(metaData)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal(file, &meta)
	if err != nil {
		return 0, err
	}
	for _, item := range meta {
		if item.ResourceName == resourceName {
			return item.MetaData.PID, nil // Found it!
		}
	}
	return 0, err
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	return false
}

func updateMeta(resourceName string) error {
	var meta []META
	data, err := os.ReadFile(metaData)
	if err == nil && len(data) > 0 {
		if err = json.Unmarshal(data, &meta); err != nil {
			return fmt.Errorf("failed to parse existing JSON: %w", err)
		}
	}
	for _, item := range meta {
		if item.ResourceName == resourceName {
			item.MetaData.Status = "stopped"
		}
	}
	updateData, err := json.MarshalIndent(meta, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	err = os.WriteFile(metaData, updateData, 0666)
	if err != nil {
		return fmt.Errorf("failed to write data to %s: %w", metaData, err)
	}
	return nil
}

//
// List VMs
//

func (vm *VMManager) ListResources() error {
	var meta []META
	var list []VmListing
	data, err := os.ReadFile(metaData)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &meta)
	if err != nil {
		return err
	}

	for _, item := range meta {
		vmDetails := VmListing{
			Name:   item.ResourceName,
			PID:    item.MetaData.PID,
			Image:  item.MetaData.Image,
			Status: item.MetaData.Status,
		}
		list = append(list, vmDetails)
	}
	// 4. Print a beautiful formatted table
	fmt.Println("======================================================================")
	// %-25s means "string format, left-aligned, padded to 25 characters"
	fmt.Printf("%-25s %-15s %-15s %-15s\n", "VM NAME", "PID", "OS IMAGE", "STATUS")
	fmt.Println("======================================================================")

	for _, v := range list {
		fmt.Printf("%-25s %-15d %-15s %-15s\n", v.Name, v.PID, v.Image, v.Status)
	}
	fmt.Println("======================================================================")

	return nil
}
