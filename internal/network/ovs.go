package network

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func (o *OvsBridgeManager) CheckBridgeExist(bridgeName string) (bool, error) {
	cmd := exec.Command(
		"ovs-vsctl", "br-exists",
		bridgeName,
	)

	err := cmd.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if exitError.ExitCode() == 2 {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (o *OvsBridgeManager) CreateBridge(bridgeName string) error {
	cmd := exec.Command(
		"ovs-vsctl", "add-br",
		bridgeName,
	)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (o *OvsBridgeManager) AttachTapDevToBridge(bridgeName, ifaceName string) error {
	cmd := exec.Command(
		"ovs-vsctl", "add-port", bridgeName, ifaceName,
	)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to attach tap device to bridge %v, %v\n", bridgeName, err)
	}
	return nil
}

func (o *OvsBridgeManager) DeleteTapFromBridge(tapName string) error {
	// 1. Fetch the bridge associated with the TAP interface
	brCmd := exec.Command("ovs-vsctl", "port-to-br", tapName)
	out, err := brCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to find bridge for port %s: %s (%w)", tapName, strings.TrimSpace(string(out)), err)
	}

	// 2. Clean the output string by trimming trailing newlines/spaces
	bridgeName := strings.TrimSpace(string(out))

	// 3. Remove the TAP interface from the identified OVS bridge
	delCmd := exec.Command("ovs-vsctl", "del-port", bridgeName, tapName)
	if delOut, err := delCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete port %s from bridge %s: %s (%w)", tapName, bridgeName, strings.TrimSpace(string(delOut)), err)
	}

	fmt.Printf("Successfully removed TAP %s from OVS bridge %s.\n", tapName, bridgeName)
	return nil
}

// func (p *TapDevManager) CheckTapBridgePortExist(ifaceName string) (bool, error) {
// 	cmd := exec.Command(
// 		"ovs-vsctl",
// 		"port-to-br",
// 		ifaceName,
// 	)
// 	err := cmd.Run()
// 	if err != nil {
// 		return false, nil
// 	}
// 	return true, nil
// }

// func (p *TapDevManager) CreateTapBridgePort(bridgeName, ifaceName string) error {
// 	cmd := exec.Command(
// 		"ovs-vsctl",
// 		"add-port",
// 		bridgeName,
// 		ifaceName,
// 		"--", "set",
// 		"interface",
// 		ifaceName,
// 		"type=internal",
// 	)
// 	fmt.Println("cmd: ", cmd)

// 	err := cmd.Run()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (p *TapDevManager) GenerateTapDevName(index, resourceId int) string {
// 	return fmt.Sprintf("tap%d%d", resourceId, index)
// }

// func (p *TapDevManager) DeleteTapFromBridgeAndSystem(ifaceName string) error {
// 	cmd := exec.Command(
// 		"ovs-vsctl", "port-to-br", ifaceName,
// 	)
// 	bridgeName, err := cmd.Output()
// 	if err != nil {
// 		return err
// 	}

// 	cmd = exec.Command(
// 		"ovs-vsctl", "del-port",
// 		string(bridgeName), ifaceName,
// 	)

// 	err = cmd.Run()
// 	if err != nil {
// 		return err
// 	}

//		return nil
//	}
func (o *OvsBridgeManager) DeleteBridgeByName(bridgeName string) error {
	cmd := exec.Command(
		"ovs-vsctl", "del-br",
		bridgeName,
	)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
