package main

import (
	"fmt"
	"incubator/internal/images"
	"incubator/internal/qemu"
	"os"

	"github.com/spf13/cobra"
)

const version = "beta-v-01"

var createVMOpts qemu.VM

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(createVMCmd)
	rootCmd.AddCommand(launchCmd)

	createVMCmd.Flags().StringVarP(&createVMOpts.Name, "name", "n", "", "Name of resource")
	createVMCmd.Flags().IntVarP(&createVMOpts.CPUs, "cpu", "c", 1, "CPU")
	createVMCmd.Flags().IntVarP(&createVMOpts.MemoryMB, "memory", "m", 1024, "Memory")
	createVMCmd.Flags().IntVarP(&createVMOpts.SSHPort, "ssh_port", "s", 2022, "SSH Port")
	createVMCmd.Flags().IntVarP(&createVMOpts.DiskSize, "disk", "d", 10, "Disk Size In G")
	createVMCmd.Flags().StringVarP(&createVMOpts.Image, "image", "i", "", "OS Image")
	createVMCmd.Flags().StringVar(&createVMOpts.CloudInitFile, "cloudinit", "", "Cloud Init File")

	createVMCmd.MarkFlagRequired("name")
	createVMCmd.MarkFlagRequired("image")

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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
		diskName := fmt.Sprintf("%s.qcow2", vmName)

		createVMOpts = qemu.VM{
			Name:          vmName,
			MemoryMB:      2048,
			CPUs:          1,
			SSHPort:       5022,
			DiskSize:      20,
			CloudInitFile: "/var/incubator/cloudinit/default.yaml",
			DiskPath:      diskName,
			Image:         args[0],
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
