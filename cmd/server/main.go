package main

import (
	"context"
	"fmt"
	"incubator/internal/logger"
	"incubator/internal/model"
	"incubator/internal/orchastrator"
	"incubator/internal/storage"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const version = "beta-v-02"
const logFile = "/var/log/incubator/app_logs.json"

var createVMOpts model.VM
var db *storage.DB
var ifaces []string
var bridgeName string
var bridgeType string

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(shutdownCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)

	createCmd.AddCommand(bridgeCmd)
	createCmd.AddCommand(vmCreateCmd)

	destroyCmd.AddCommand(vmDestroyCmd)

	listCmd.AddCommand(listVmCmd)

	vmCreateCmd.Flags().StringVar(&createVMOpts.Name, "name", "auto", "VM name")
	vmCreateCmd.Flags().IntVar(&createVMOpts.CPUs, "cpu", 2, "Number of vCPUs")
	vmCreateCmd.Flags().IntVarP(&createVMOpts.MemoryMB, "memory", "m", 1024, "Memory size (MB)")
	vmCreateCmd.Flags().IntVar(&createVMOpts.DiskSize, "disk", 10, "Disk size (GB)")
	vmCreateCmd.Flags().StringVar(&createVMOpts.Image, "image", "ubuntu", "VM image/template")
	vmCreateCmd.Flags().StringVar(&createVMOpts.Version, "version", "jammy", "os version")
	vmCreateCmd.Flags().StringSliceVar(&ifaces, "iface", []string{"default"}, "Network Bridges")

	bridgeCmd.Flags().StringVar(&bridgeName, "name", "", "Bridge Name")
	bridgeCmd.Flags().StringVar(&bridgeType, "type", "linux-bridge", "Bridge Type")
	bridgeCmd.MarkFlagRequired("name")

}

func main() {
	db = storage.InitDB()
	if db == nil {
		log.Fatalf("Error Connecting to  DB")
	}

	defer db.Cli.Close()

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}

	defer file.Close()

	log.SetOutput(file)
	log.SetFlags(0)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

var rootCmd = &cobra.Command{
	Use:   "incubator",
	Short: "Incubator: A Tiny Openstack",
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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Resource",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "create a bridge",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {

		return nil
	},
}

var vmCreateCmd = &cobra.Command{
	Use:   "vm",
	Short: "create a vm",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(createVMOpts.Name) < 3 {
			return fmt.Errorf("Name must have atleats 4 character")
		}

		taskId := logger.GenerateTaskId()
		ctx := context.WithValue(context.Background(), logger.TaskIdKey, taskId)

		o := orchastrator.VMManager(db, createVMOpts)
		o.Iface = ifaces
		err := o.CreateVMHandler(ctx)
		if err != nil {
			return err
		}

		fmt.Println("VM created successfully!")
		return nil
	},
}

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy Resource",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var vmDestroyCmd = &cobra.Command{
	Use:          "vm",
	Short:        "destroy a vm",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskId := logger.GenerateTaskId()
		ctx := context.WithValue(context.Background(), logger.TaskIdKey, taskId)

		o := orchastrator.VMManager(db, createVMOpts)

		resourceId := args[0]
		err := o.DestroyVMHandler(ctx, resourceId)
		if err != nil {
			return fmt.Errorf("qemu failed to destroy resource: %w\n", err)
		}
		fmt.Printf("Succesfully Destroyed Resource, %s\n", resourceId)
		return nil
	},
}

var shutdownCmd = &cobra.Command{
	Use:   "stop",
	Short: "Poweroff Resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskId := logger.GenerateTaskId()
		ctx := context.WithValue(context.Background(), logger.TaskIdKey, taskId)

		o := orchastrator.VMManager(db, createVMOpts)
		resourceId := args[0]
		err := o.ShutdownVMHandler(ctx, resourceId)
		if err != nil {
			return fmt.Errorf("qemu failed to stop vm: %w", err)
		}
		fmt.Println("VM stopped successfully!")
		return nil

	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskId := logger.GenerateTaskId()
		ctx := context.WithValue(context.Background(), logger.TaskIdKey, taskId)

		o := orchastrator.VMManager(db, createVMOpts)
		resourceId := args[0]
		err := o.StartVMHandler(ctx, resourceId)
		if err != nil {
			return fmt.Errorf("qemu failed to start resource: %w\n", err)
		}
		fmt.Printf("Succesfully Started Resource, %s\n", resourceId)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "List Resources",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
}

var listVmCmd = &cobra.Command{
	Use:          "vm",
	Short:        "List Resources",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskId := logger.GenerateTaskId()
		ctx := context.WithValue(context.Background(), logger.TaskIdKey, taskId)

		o := orchastrator.VMManager(db, createVMOpts)
		var resourceId string
		if len(args) > 0 {
			resourceId = args[0]
		}
		err := o.ListResources(ctx, resourceId)
		if err != nil {
			return fmt.Errorf("qemu failed to list resources: %w", err)
		}
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "show resource status",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskId := logger.GenerateTaskId()
		ctx := context.WithValue(context.Background(), logger.TaskIdKey, taskId)

		o := orchastrator.VMManager(db, createVMOpts)
		var resourceId string
		if len(args) > 0 {
			resourceId = args[0]
		}
		err := o.VmStatusHandler(ctx, resourceId)
		if err != nil {
			return fmt.Errorf("qemu failed to list resources: %w", err)
		}
		return nil
	},
}

// var launchCmd = &cobra.Command{
// 	Use:   "launch",
// 	Short: "Launch Resourse",
// 	Args:  cobra.ExactArgs(1),
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		images := images.StoreImages()
// 		vmName := qemu.GenerateVMName()
// 		// diskName := fmt.Sprintf("%s.qcow2", vmName)
// 		diskName := filepath.Join(diskStore, vmName, vmName+".qcow2")

// 		createVMOpts = qemu.VM{
// 			Name:          vmName,
// 			MemoryMB:      2048,
// 			CPUs:          1,
// 			DiskSize:      20,
// 			CloudInitFile: "/var/incubator/cloudinit/default.yaml",
// 			DiskPath:      diskName,
// 			Image:         args[0],
// 			Status:        "running",
// 		}
// 		m := qemu.NewVMManager()
// 		err := m.CreateVM(createVMOpts, images)
// 		if err != nil {
// 			return fmt.Errorf("qemu failed to create vm: %w", err)
// 		}

// 		fmt.Println("VM created successfully!")
// 		return nil
// 	},
// }

// var shutdownCmd = &cobra.Command{
// 	Use:   "stop",
// 	Short: "Poweroff Resource",
// 	Args:  cobra.ExactArgs(1),
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		resourceName := args[0]
// 		m := qemu.NewVMManager()
// 		err := m.ShutdownVM(resourceName)
// 		if err != nil {
// 			return fmt.Errorf("qemu failed to stop vm: %w", err)
// 		}
// 		fmt.Println("VM stopped successfully!")
// 		return nil

// 	},
// }
