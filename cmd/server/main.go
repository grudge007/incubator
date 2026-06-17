package main

import (
	"fmt"
	"incubator/internal/images"
	"incubator/internal/qemu"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const version = "beta-v-01"
const diskStore = "/var/incubator/disk/"

var createVMOpts qemu.VM

// var ShutdownVMOpts qemu.VM

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(createVMCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(shutdownCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(startCmd)

	createVMCmd.Flags().StringVarP(&createVMOpts.Name, "name", "n", "", "Name of resource")
	createVMCmd.Flags().IntVarP(&createVMOpts.CPUs, "cpu", "c", 1, "CPU")
	createVMCmd.Flags().IntVarP(&createVMOpts.MemoryMB, "memory", "m", 1024, "Memory")
	createVMCmd.Flags().IntVarP(&createVMOpts.VNC, "vnc", "v", 2022, "VNC Port")
	createVMCmd.Flags().IntVarP(&createVMOpts.DiskSize, "disk", "d", 10, "Disk Size In G")
	createVMCmd.Flags().StringVarP(&createVMOpts.Image, "image", "i", "", "OS Image")
	createVMCmd.Flags().StringVar(&createVMOpts.CloudInitFile, "cloudinit", "", "Cloud Init File")

	createVMCmd.MarkFlagRequired("name")
	createVMCmd.MarkFlagRequired("image")

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// fmt.Println(err)
		os.Exit(1)
	}

}

var rootCmd = &cobra.Command{
	Use:   "incubator",
	Short: "a tiny openstack",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to incubator! Use --help to see available commands.")
	},
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

var createVMCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource",
	RunE: func(cmd *cobra.Command, args []string) error {
		images := images.StoreImages()
		m := qemu.NewVMManager()
		createVMOpts.DiskPath = createVMOpts.Name + ".qcow2"
		err := m.CreateVM(createVMOpts, images)
		if err != nil {
			return fmt.Errorf("qemu failed to create vm: %w", err)
		}

		fmt.Println("VM created successfully!")
		return nil
	},
}

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch Resourse",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		images := images.StoreImages()
		vmName := qemu.GenerateVMName()
		// diskName := fmt.Sprintf("%s.qcow2", vmName)
		diskName := filepath.Join(diskStore, vmName, vmName+".qcow2")

		createVMOpts = qemu.VM{
			Name:          vmName,
			MemoryMB:      2048,
			CPUs:          1,
			DiskSize:      20,
			CloudInitFile: "/var/incubator/cloudinit/default.yaml",
			DiskPath:      diskName,
			Image:         args[0],
			Status:        "running",
		}
		m := qemu.NewVMManager()
		err := m.CreateVM(createVMOpts, images)
		if err != nil {
			return fmt.Errorf("qemu failed to create vm: %w", err)
		}

		fmt.Println("VM created successfully!")
		return nil
	},
}

var shutdownCmd = &cobra.Command{
	Use:   "stop",
	Short: "Poweroff Resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceName := args[0]
		m := qemu.NewVMManager()
		err := m.ShutdownVM(resourceName)
		if err != nil {
			return fmt.Errorf("qemu failed to stop vm: %w", err)
		}
		fmt.Println("VM stopped successfully!")
		return nil

	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := qemu.NewVMManager()
		err := m.ListResources()
		if err != nil {
			return fmt.Errorf("qemu failed to list resources: %w", err)
		}
		return nil
	},
}

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy Resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m := qemu.NewVMManager()
		resourceName := args[0]
		err := m.DestroyResource(resourceName)
		if err != nil {
			return fmt.Errorf("qemu failed to destroy resource: %w\n", err)
		}
		fmt.Printf("Succesfully Destroyed Resource, %s\n", resourceName)
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m := qemu.NewVMManager()
		resourceName := args[0]
		err := m.StartVM(resourceName)
		if err != nil {
			return fmt.Errorf("qemu failed to start resource: %w\n", err)
		}
		fmt.Printf("Succesfully Started Resource, %s\n", resourceName)
		return nil
	},
}
