package validation

import (
	"fmt"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

var rfc1918Ranges = []*net.IPNet{
	mustParseCIDR("10.0.0.0/8"),
	mustParseCIDR("172.16.0.0/12"),
	mustParseCIDR("192.168.0.0/16"),
}

func mustParseCIDR(c string) *net.IPNet {
	_, n, _ := net.ParseCIDR(c)
	return n
}

func CidrIntersects(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

func CidrInRFC1918(cidr *net.IPNet) bool {
	for _, r := range rfc1918Ranges {
		if r.Contains(cidr.IP) {
			return true
		}
	}
	return false
}

func IsNetworkLargeEnough(cidr *net.IPNet, minMaskSize int) bool {
	maskSize, _ := cidr.Mask.Size()
	return maskSize <= minMaskSize
}

func ValidateCIDRInRanges(v interface{}, path cty.Path) diag.Diagnostics {
	str := v.(string)

	_, cidr, err := net.ParseCIDR(str)
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Incorrect CIDR",
				Detail:   fmt.Sprintf("Value '%s' is not a valid CIDR", str),
			},
		}
	}

	if !IsNetworkLargeEnough(cidr, 18) {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Incorrect CIDR",
				Detail:   fmt.Sprintf("chosen subnet %s is too small, subnet must be larger than /18", str),
			},
		}
	}

	if !CidrInRFC1918(cidr) {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "CIDR not in allowed private ranges",
				Detail: fmt.Sprintf(
					"CIDR %s must be inside RFC1918 ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16",
					str,
				),
			},
		}
	}

	return nil
}
