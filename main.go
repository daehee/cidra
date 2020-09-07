package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"

	"inet.af/netaddr"
)

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	var err error

	var m i2aMap
	// load ip2asn database
	err = m.new()
	check(err)

	// initialize channels
	ipChan := make(chan string)
	outChan := make(chan []string)

	// initialize map for atomic counter and metadata
	cidrs := make(map[string]*asnData)

	var outWG sync.WaitGroup
	outWG.Add(1)
	go output(&outWG, outChan)

	var ipWG sync.WaitGroup
	ipWG.Add(1)
	go func() {
		for ipStr := range ipChan {
			ip, err := netaddr.ParseIP(ipStr)
			// Skip if invalid IP
			if err != nil {
				continue
			}

			// lookup ASN of IP address
			asn := m.ASofIP(ip)
			// skip if invalid ASN returning 0 value
			if asn == 0 {
				continue
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
				if tmpNet.Contains(net.ParseIP(ipStr)) {
					cidr = tmpNet.String()
					continue
				}
			}
			if cidr == "" {
				log.Fatalln("CIDR not found")
			}

			// increment counter if previously parsed and skip rest of metadata lookup & assignment operations
			if counter, ok := cidrs[cidr]; ok {
				counter.addCount()
				continue
			}

			desc := m.ASName(asn)

			cidrs[cidr] = &asnData{
				asn:  asn,
				desc: desc,
			}
			// utilize atomic counter
			cidrs[cidr].addCount()

		}

		// sort descending by CIDR counts
		p := make(CIDRCountsList, len(cidrs))
		i := 0
		for cidr, data := range cidrs {
			p[i] = CIDRCounts{cidr, data.n}
			i++
		}
		sort.Sort(sort.Reverse(p))
		for _, k := range p {
			outChan <- []string{fmt.Sprintf("AS%d", cidrs[k.Key].asn), cidrs[k.Key].desc, k.Key, strconv.Itoa(cidrs[k.Key].getCount())}
		}

		ipWG.Done()
	}()

	go func() {
		ipWG.Wait()
		close(outChan)
	}()

	// read list of IPs from stdin
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		ipStr := sc.Text()
		// TODO validate and clean IP before sending into channel
		ipChan <- ipStr
	}
	close(ipChan)

	outWG.Wait()
}
