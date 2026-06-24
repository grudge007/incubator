package network

import "fmt"

type BridgeManager interface {
	CheckBridgeExist(bridgeName string) (bool, error)
	CreateBridge(bridgeName string) error
	CheckTapBridgePortExist(ifaceName string) (bool, error)
	CreateTapBridgePort(bridgeName, ifaceName string) error
	GenerateTapDevName(index, resourceId int) string
	DeleteTapFromBridgeAndSystem(ifaceName string) error
	DeleteBridgeByName(bridgeName string) error
}

// type BridgeManager interface {
//     CheckBridgeExist(name string) (bool, error)
//     CreateBridge(name string) error
//     DeleteBridgeByName(name string) error
// }

// type PortManager interface {
//     CheckTapBridgePortExist(name string) (bool, error)
//     CreateTapBridgePort(bridgeName string, ifaceName string) error
//     DeleteTapFromBridgeAndSystem(tapName string) error
//     GenerateTapDevName(i, resId int) string
// }

type LinuxBridgeManager struct{}

type OvsBridgeManager struct{}

func NewLinuxBridgeManager() *LinuxBridgeManager {
	return &LinuxBridgeManager{}
}

func NewOvsBridgeManager() *OvsBridgeManager {
	return &OvsBridgeManager{}
}

func NewBridgeManager(bridgeType string) (BridgeManager, error) {
	switch bridgeType {

	case "linux-bridge":
		return NewLinuxBridgeManager(), nil

	case "ovs":
		return NewOvsBridgeManager(), nil

	default:
		return nil, fmt.Errorf("unsupported bridge type, %v", bridgeType)
	}
}
