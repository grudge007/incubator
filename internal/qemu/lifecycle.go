package qemu

import (
	"fmt"
	"incubator/internal/images"
	"os/exec"
	"strconv"
)

const ImagePath = "/var/incubator/images/"

type VM struct {
	Name          string
	PID           int
	CPUs          int
	MemoryMB      int
	SSHPort       int
	DiskSize      int
	DiskPath      string
	CloudInitFile string
	Image         string
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

func (vm *VMManager) CreateVM(vmDetails VM, availableImages images.AvailableImages) error {
	memStr := strconv.Itoa(vmDetails.MemoryMB)
	cpuStr := strconv.Itoa(vmDetails.CPUs)
	sshFwd := fmt.Sprintf("tcp::%d-:22", vmDetails.SSHPort)
	driveArg := fmt.Sprintf("file=%s.qcow2,format=qcow2,media=disk", vmDetails.Name)
	diskName := vmDetails.DiskPath

	imageFile, err := findImageFile(vmDetails.Image, availableImages)
	if err != nil {
		return err
	}
	err = vm.createVMDisk(imageFile, diskName)
	if err != nil {
		return err
	}

	err = vm.resizeVMDisk(diskName, vmDetails.DiskSize)
	if err != nil {
		return err
	}
	fmt.Println(vmDetails.CloudInitFile)

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
		"-nographic",
		// "-display", "none", // Disables the local QEMU window on the host
		// "-vnc", "0.0.0.0:1",
	)

	err = cmd.Start()
	if err != nil {
		return err
	}
	vmDetails.PID = cmd.Process.Pid
	fmt.Println("PID: ", vmDetails.PID)
	return nil

}

func (vm *VMManager) createVMDisk(imageFile, diskName string) error {
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

//	func (vm *VMManager) createCloudInit() {
//		config := CloudInit{
//			Users: []User{
//				{
//					Name: ,
//				}
//			},
//		}
//	}
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
