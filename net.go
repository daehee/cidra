package main

import (
	"errors"
	"strings"
	"sync"

	"github.com/bradfitz/ip2asn"
	"github.com/daehee/ip2cidr"
	"inet.af/netaddr"
)

type netMap struct {
	*ip2asn.Map
	rm sync.RWMutex
}

func (m *netMap) new() error {
	var err error
	m.Map, err = ip2asn.OpenFile("ip2asn-combined.tsv.gz")
	if err != nil {
		return err
	}
	return nil
}

func (m *netMap) ip2ASNCIDR(ip *netaddr.IP) (string, int, error) {
	asn, ipr := m.ASRange(*ip)
	if asn == 0 {
		return "", 0, errors.New("ASN not found")
	}

	cidrs, err := ip2cidr.IPRangeToCIDR(ipr[0], ipr[1])
	if len(cidrs) < 1 || err != nil {
		return "", 0, errors.New("no matching CIDRs")
	}

	// test for containing cidr
	var cidr string
	for _, c := range cidrs {
		p, err := netaddr.ParseIPPrefix(c)
		if err != nil {
			continue
		}
		if ok := p.Contains(*ip); ok {
			cidr = c
		}
	}

	if cidr == "" {
		return "", 0, errors.New("no matching CIDR")
	}
	return cidr, asn, nil
}

// https://stackoverflow.com/a/48519490
func IsIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}
