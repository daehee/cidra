package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"text/tabwriter"

	"inet.af/netaddr"
)

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type asnData struct {
	n    int32
	cidr string
	desc string
}

func (c *asnData) AddCount() {
	atomic.AddInt32(&c.n, 1)
}

func (c *asnData) GetCount() int {
	return int(atomic.LoadInt32(&c.n))
}

func main() {
	var err error

	var m asnMap
	// load ip2asn database
	err = m.openDB()
	check(err)

	// initialize channels
	ips := make(chan string)
	output := make(chan string)

	// initialize map for atomic counter and metadata
	asns := make(map[int]*asnData)

	var outputWG sync.WaitGroup
	outputWG.Add(1)
	go func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)

		for o := range output {
			_, err := fmt.Fprintln(w, o)
			check(err)
		}

		w.Flush()
		outputWG.Done()
	}()

	var ipWG sync.WaitGroup
	ipWG.Add(1)
	go func() {
		for ipStr := range ips {
			ip, err := netaddr.ParseIP(ipStr)
			// Skip if invalid IP
			if err != nil {
				continue
			}

			asn := m.ASofIP(ip)
			// skip if invalid ASN returning 0 value
			if asn == 0 {
				continue
			}
			// only increment counter if previously parsed
			if counter, ok := asns[asn]; ok {
				counter.AddCount()
				continue
			}

			// convert IP range to CIDR
			ipRange := m.ASRange(asn)
			cidr := rangeCIDR(net.ParseIP(ipRange[0]), net.ParseIP(ipRange[1]))

			desc := m.ASName(asn)

			// assign ASN data to map struct
			asns[asn] = &asnData{
				cidr: cidr.String(),
				desc: desc,
			}
			// utilize atomic counter
			asns[asn].AddCount()

		}

		// sort descending by ASN counts
		p := make(ASNCountsList, len(asns))
		i := 0
		for asn, data := range asns {
			p[i] = ASNCounts{asn, data.n}
			i++
		}
		sort.Sort(sort.Reverse(p))
		for _, k := range p {
			output <- fmt.Sprintf("%d\t%s\t%s\t%d", k.Key, asns[k.Key].desc, asns[k.Key].cidr, asns[k.Key].GetCount())
		}

		ipWG.Done()
	}()

	go func() {
		ipWG.Wait()
		close(output)
	}()

	// Read list of IPs from stdin
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		ipStr := sc.Text()
		ips <- ipStr
	}
	close(ips)

	outputWG.Wait()
}
