package main

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/bradfitz/ip2asn"
	"inet.af/netaddr"
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

func (m *i2aMap) ipCIDR(ip *netaddr.IP) (string, int, error) {
	// lookup ASN of IP address
	asn := m.ASofIP(*ip)
	// skip if invalid ASN returning 0 value
	if asn == 0 {
		return "", 0, errors.New("AS0")
	}

	// TODO optimize: first check if there's a match in cached map of ASNs to CIDRs before diving into more expensive operations

	// lookup specific CIDR for this IP
	// IP -> ASN -> CIDRs -> CIDR containing IP
	var cidr string
	ipRanges := m.ASRanges(asn)
	for _, r := range ipRanges {
		// skip if IPv6 since rangeCIDR function currently not compatible
		if !IsIPv4(r.StartIP) || !IsIPv4(r.EndIP) {
			continue
		}
		tmpNet := rangeCIDR(net.ParseIP(r.StartIP), net.ParseIP(r.EndIP))
		// ASN has multiple ranges, so check which range this IP belongs to
		if tmpNet.Contains(net.ParseIP(ip.String())) {
			cidr = tmpNet.String()
			continue
		}
	}
	return cidr, asn, nil
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
