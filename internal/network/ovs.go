package network

import (
	"errors"
	"fmt"
	"os/exec"
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

func (o *OvsBridgeManager) CheckTapBridgePortExist(ifaceName string) (bool, error) {
	cmd := exec.Command(
		"ovs-vsctl",
		"port-to-br",
		ifaceName,
	)
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (o *OvsBridgeManager) CreateTapBridgePort(bridgeName, ifaceName string) error {
	cmd := exec.Command(
		"ovs-vsctl",
		"add-port",
		bridgeName,
		ifaceName,
		"--", "set",
		"interface",
		ifaceName,
		"tpye=internal",
	)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (o *OvsBridgeManager) GenerateTapDevName(index, resourceId int) string {
	return fmt.Sprintf("tap%d%d", resourceId, index)
}

func (o *OvsBridgeManager) DeleteTapFromBridgeAndSystem(ifaceName string) error {
	return nil
}
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
