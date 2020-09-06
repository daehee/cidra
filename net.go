package main

import (
    "fmt"
    "net"

    "github.com/bradfitz/ip2asn"
)

type asnMap struct {
    *ip2asn.Map
}

func (m *asnMap) openDB() error {
    var err error
    m.Map, err = ip2asn.OpenFile("ip2asn-combined.tsv.gz")
    if err != nil {
        return err
    }
    return nil
}

func (m *asnMap) getCIDR(asn int) *net.IPNet {
    if asn == 0 { return nil }
    ipRange := m.ASRange(asn)
    cidr := rangeCIDR(net.ParseIP(ipRange[0]), net.ParseIP(ipRange[1]))
    return cidr
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
    _, cidr, err := net.ParseCIDR(cidrStr)
    check(err)

    return cidr
}

