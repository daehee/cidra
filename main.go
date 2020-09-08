package main

import (
	"bufio"
	"fmt"
	"log"
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

	// initialize map for atomic counter and metadata
	cidrs := make(map[string]*asnData)

	// initialize channels
	ipChan := make(chan string)
	outChan := make(chan []string)

	var outWG sync.WaitGroup
	outWG.Add(1)
	go output(&outWG, outChan)

	// TODO move sort to separate stage of pipeline
	// TODO fan out heavy-load processing with multiple workers
	var ipWG sync.WaitGroup
	ipWG.Add(1)
	go func() {
		for ipStr := range ipChan {
			ip, err := netaddr.ParseIP(ipStr)
			// Skip if invalid IP
			if err != nil {
				continue
			}

			// lookup CIDR and ASN number for this IP
			cidr, asn, err := m.ipCIDR(&ip)
			if err != nil {
				continue
			}

			// increment counter if previously parsed and skip rest of metadata lookup & assignment operations
			if counter, ok := cidrs[cidr]; ok {
				counter.addCount()
				continue
			}

			// add new map entry for CIDR
			cidrs[cidr] = &asnData{
				asn:  asn,
				desc: m.ASName(asn),
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
		ipChan <- ipStr
	}
	close(ipChan)

	outWG.Wait()
}
