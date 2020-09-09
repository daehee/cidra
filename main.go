package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"inet.af/netaddr"
)

var (
	cpu = runtime.NumCPU()
)

func inGen(c chan<- string, done <-chan interface{}) {
	defer close(c)
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		select {
		case <-done:
			return
		case c <- sc.Text():
		}
	}
}

type result struct {
	cidr string
	asn  int
	err  error
}

type count struct {
	n int32
	d result
}

func (c *count) add() {
	atomic.AddInt32(&c.n, 1)
}

func (c *count) get() int {
	return int(atomic.LoadInt32(&c.n))
}

func lookup(m *netMap, i <-chan string, done <-chan interface{}) <-chan result {
	c := make(chan result)
	go func() {
		defer close(c)
		for ip := range i {
			ip, err := netaddr.ParseIP(ip)
			if err != nil {
				continue // skip invalid IP
			}
			cidr, asn, err := m.ip2ASNCIDR(&ip)
			if err != nil {
				continue // skip invalid
			}

			res := result{cidr: cidr, asn: asn, err: err}
			select {
			case <-done:
				return
			case c <- res:
			}
		}
	}()
	return c
}

// counter merges channels receiving results from lookup workers
func counter(c chan<- count, done <-chan interface{}, ics ...<-chan result) {
	var wg sync.WaitGroup
	seen := make(map[string]*count) // map of unique CIDRs & counts
	var sm sync.RWMutex             // seen mutex

	wg.Add(len(ics))
	multiplex := func(ic <-chan result) {
		defer wg.Done()
		for i := range ic {
			select {
			case <-done:
				return
			default:
				sm.Lock()
				counter, found := seen[i.cidr]
				sm.Unlock()
				if found {
					counter.add()
					continue
				}

				sm.Lock()
				seen[i.cidr] = &count{d: i}
				seen[i.cidr].add()
				sm.Unlock()
			}
		}
	}

	for _, ic := range ics {
		go multiplex(ic)
	}

	go func() {
		wg.Wait() // wait for merge multiplexing complete
		for _, v := range seen {
			select {
			case <-done:
				return
			case c <- *v: // send completed count struct
			}
		}
		close(c)
	}()
}

func main() {
	runtime.GOMAXPROCS(cpu)

	done := make(chan interface{})
	defer close(done)

	ips := make(chan string)
	go inGen(ips, done)

	var nm netMap
	if err := nm.new(); err != nil {
		log.Fatal(err)
	}

	// fan out CIDR lookups to workers
	cidrs := make([]<-chan result, cpu)
	for i := 0; i < cpu; i++ {
		cidrs[i] = lookup(&nm, ips, done)
	}

	counts := make(chan count)
	// fan in CIDR results to counter
	go counter(counts, done, cidrs...)

	// TODO next stage: sorter

	w := csv.NewWriter(os.Stdout)
	for v := range counts {
		o := []string{fmt.Sprintf("AS%d", v.d.asn), nm.ASName(v.d.asn), v.d.cidr, strconv.Itoa(v.get())}
		if err := w.Write(o); err != nil {
			log.Fatal(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}
