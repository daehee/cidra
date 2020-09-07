package main

type CIDRCounts struct {
	Key   string
	Value int32
}

type CIDRCountsList []CIDRCounts

func (p CIDRCountsList) Len() int           { return len(p) }
func (p CIDRCountsList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p CIDRCountsList) Less(i, j int) bool { return p[i].Value < p[j].Value }
