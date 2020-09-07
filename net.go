package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/bradfitz/ip2asn"
)

type i2aMap struct {
	*ip2asn.Map
}

func (m *i2aMap) new() error {
	var err error
	m.Map, err = ip2asn.OpenFile("ip2asn-combined.tsv.gz")
	if err != nil {
		return err
	}
	return nil
}

// TODO improve accuracy with more granular multiple CIDR output
// Credit: https://groups.google.com/g/golang-nuts/c/rJvVwk4jwjQ/m/4eCBR_HwrhQJ
func rangeCIDR(first, last net.IP) *net.IPNet {
	var l int
	var tmpIP net.IP
	maxLen := 32

	for l = maxLen; l >= 0; l-- {
		mask := net.CIDRMask(l, maxLen)
		tmpIP = first.Mask(mask)
		tmpNet := net.IPNet{IP: tmpIP, Mask: mask}

		if tmpNet.Contains(last) {
			break
		}
	}

	cidrStr := fmt.Sprintf("%v/%v", tmpIP, l)
	_, cidr, _ := net.ParseCIDR(cidrStr)
	return cidr
}

// https://stackoverflow.com/a/48519490
func IsIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}
