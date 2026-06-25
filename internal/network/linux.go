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

func (l *LinuxBridgeManager) AttachTapDevToBridge(bridgeName string, tapIfaceName string) error {
	// 1. Fetch the existing bridge link by its name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("failed to find bridge %s: %w", bridgeName, err)
	}

	// 2. Fetch the existing TAP interface link by its name
	tapLink, err := netlink.LinkByName(tapIfaceName)
	if err != nil {
		return fmt.Errorf("failed to find TAP interface %s: %w", tapIfaceName, err)
	}

	// 3. Attach the TAP interface to the bridge (Set bridge as master)
	if err := netlink.LinkSetMaster(tapLink, br); err != nil {
		// Clean up the TAP link if attachment fails
		_ = netlink.LinkDel(tapLink)
		return fmt.Errorf("failed to attach TAP %s to bridge %s: %w", tapIfaceName, bridgeName, err)
	}

	// 4. Bring the TAP interface UP
	if err := netlink.LinkSetUp(tapLink); err != nil {
		return fmt.Errorf("failed to bring TAP %s UP: %w", tapIfaceName, err)
	}

	fmt.Printf("Successfully attached TAP %s to bridge %s and brought it UP.\n", tapIfaceName, bridgeName)
	return nil
}

func (l *LinuxBridgeManager) DeleteTapFromBridge(tapName string) error {
	// 1. Fetch the TAP interface link
	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		// If it already doesn't exist, we can treat this as a success or return early
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			fmt.Printf("TAP interface %s does not exist. Nothing to delete -- lxbr.\n", tapName)
			return nil
		}
		return fmt.Errorf("failed to find TAP interface %s: %w", tapName, err)
	}

	if err := netlink.LinkSetNoMaster(tapLink); err != nil {
		return fmt.Errorf("failed to detach TAP %s from its bridge: %w", tapName, err)
	}
	fmt.Printf("Successfully detached %s from the bridge.\n", tapName)

	// 3. DELETE THE TAP INTERFACE ENTIRELY
	// This removes the interface from the Linux kernel completely.
	// if err := netlink.LinkDel(tapLink); err != nil {
	// 	return fmt.Errorf("failed to delete TAP interface %s: %w", tapName, err)
	// }
	// fmt.Printf("Successfully deleted TAP interface %s from the system.\n", tapName)

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
