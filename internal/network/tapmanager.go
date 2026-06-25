package network

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

func (p *TapDevManager) DeleteTapFromSystem(tapName string) error {
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

	if err := netlink.LinkDel(tapLink); err != nil {
		return fmt.Errorf("failed to delete TAP interface %s: %w", tapName, err)
	}
	fmt.Printf("Successfully deleted TAP interface %s from the system.\n", tapName)

	return nil
}

func (p *TapDevManager) GenerateTapDevName(i, resId int) string {
	return fmt.Sprintf("tap%d%d", resId, i)
}

func (p *TapDevManager) CheckTapBridgePortExist(ifaceName string) (bool, error) {
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

func (p *TapDevManager) CreateTapPort(ifaceName string) error {
	la := netlink.NewLinkAttrs()
	la.Name = ifaceName
	tap := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TAP,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		return fmt.Errorf("failed to create tap interface %s: %w", ifaceName, err)
	}

	// this was missing
	tapLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to find tap after creation %s: %w", ifaceName, err)
	}

	if err := netlink.LinkSetUp(tapLink); err != nil {
		return fmt.Errorf("failed to bring tap %s up: %w", ifaceName, err)
	}

	return nil
}
