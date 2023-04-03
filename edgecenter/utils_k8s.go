package edgecenter

import (
	"net"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func parseCIDRFromString(cidr string) (edgecloud.CIDR, error) {
	var ecCIDR edgecloud.CIDR
	_, netIPNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ecCIDR, err
	}
	ecCIDR.IP = netIPNet.IP
	ecCIDR.Mask = netIPNet.Mask

	return ecCIDR, nil
}
