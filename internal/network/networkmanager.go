package network

import (
	"fmt"
)

type BridgeManager interface {
	CheckBridgeExist(name string) (bool, error)
	CreateBridge(name string) error
	DeleteBridgeByName(name string) error
	AttachTapDevToBridge(bridgeName, ifaceName string) error
	DeleteTapFromBridge(tapName string) error
}

type PortManager interface {
	CheckTapBridgePortExist(name string) (bool, error)
	CreateTapPort(ifaceName string) error
	GenerateTapDevName(i, resId int) string
	DeleteTapFromSystem(tapName string) error
}

type LinuxBridgeManager struct{}

type OvsBridgeManager struct{}

type TapDevManager struct{}

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

func NewTapDevManager() *TapDevManager {
	return &TapDevManager{}
}
