package network

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

func (l *LinuxBridgeManager) CheckBridgeExist(bridgeName string) (bool, error) {
	link, err := netlink.LinkByName(bridgeName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	if link.Type() == "bridge" {
		return true, nil
	}

	return false, fmt.Errorf("interface %s exists but is type %s, not bridge", bridgeName, link.Type())
}

func (l *LinuxBridgeManager) CreateBridge(name string) error {
	// 1. Define the bridge link attributes
	la := netlink.NewLinkAttrs()
	la.Name = name

	bridge := &netlink.Bridge{LinkAttrs: la}

	// 2. Create the bridge interface
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to create bridge %s: %w", name, err)
	}

	// 3. Bring the interface UP (optional but usually required)
	if err := netlink.LinkSetUp(bridge); err != nil {
		return fmt.Errorf("failed to bring bridge %s up: %w", name, err)
	}

	fmt.Printf("Bridge %s created and set to UP successfully.\n", name)
	return nil
}

func (l *LinuxBridgeManager) CheckTapBridgePortExist(ifaceName string) (bool, error) {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return false, nil
		}
		return false, err
	}

	if link.Type() == "tuntap" {
		return true, nil
	}

	return false, fmt.Errorf("interface %s exists but is type %s, not tap", ifaceName, link.Type())

}

func (l *LinuxBridgeManager) CreateTapBridgePort(bridgeName string, ifaceName string) error {
	// 1. Fetch the existing bridge link by its name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("failed to find bridge %s: %w", bridgeName, err)
	}

	// 2. Define the TAP interface attributes
	la := netlink.NewLinkAttrs()
	la.Name = ifaceName

	tap := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TAP,
	}

	// 3. Create the TAP interface in the kernel
	if err := netlink.LinkAdd(tap); err != nil {
		return fmt.Errorf("failed to create tap interface %s: %w", ifaceName, err)
	}

	// 4. Attach the TAP interface to the bridge
	if err := netlink.LinkSetMaster(tap, br); err != nil {
		// Clean up the created link if attachment fails
		_ = netlink.LinkDel(tap)
		return fmt.Errorf("failed to attach TAP %s to bridge %s: %w", ifaceName, bridgeName, err)
	}

	// 5. Bring the TAP interface UP
	if err := netlink.LinkSetUp(tap); err != nil {
		return fmt.Errorf("failed to bring TAP %s UP: %w", ifaceName, err)
	}

	fmt.Printf("Successfully attached %s to %s and brought it UP.\n", ifaceName, bridgeName)
	return nil
}

func (l *LinuxBridgeManager) GenerateTapDevName(i, resId int) string {
	return fmt.Sprintf("tap%d%d", resId, i)
}

func (l *LinuxBridgeManager) DeleteTapFromBridgeAndSystem(tapName string) error {
	// 1. Fetch the TAP interface link
	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		// If it already doesn't exist, we can treat this as a success or return early
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			fmt.Printf("TAP interface %s does not exist. Nothing to delete.\n", tapName)
			return nil
		}
		return fmt.Errorf("failed to find TAP interface %s: %w", tapName, err)
	}

	// 2. DETACH FROM BRIDGE (brctl delif equivalent)
	// Setting the MasterIndex to 0 or passing nil to LinkSetNoMaster
	// un-enslaves the port from whatever bridge it is currently attached to.
	if err := netlink.LinkSetNoMaster(tapLink); err != nil {
		return fmt.Errorf("failed to detach TAP %s from its bridge: %w", tapName, err)
	}
	fmt.Printf("Successfully detached %s from the bridge.\n", tapName)

	// 3. DELETE THE TAP INTERFACE ENTIRELY
	// This removes the interface from the Linux kernel completely.
	if err := netlink.LinkDel(tapLink); err != nil {
		return fmt.Errorf("failed to delete TAP interface %s: %w", tapName, err)
	}
	fmt.Printf("Successfully deleted TAP interface %s from the system.\n", tapName)

	return nil
}

func (l *LinuxBridgeManager) DeleteBridgeByName(name string) error {
	// Create a bridge object with the specified name
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
	}

	// Attempt to delete the link directly
	if err := netlink.LinkDel(bridge); err != nil {
		return fmt.Errorf("failed to delete bridge %s: %w", name, err)
	}

	return nil
}
