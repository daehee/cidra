package main

import "sync/atomic"

type asnData struct {
	n    int32
	asn  int
	desc string
}

func (c *asnData) addCount() {
	atomic.AddInt32(&c.n, 1)
}

func (c *asnData) getCount() int {
	return int(atomic.LoadInt32(&c.n))
}
