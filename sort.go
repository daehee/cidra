package main

type ASNCounts struct {
    Key int
    Value int32
}

type ASNCountsList []ASNCounts

func (p ASNCountsList) Len() int           { return len(p) }
func (p ASNCountsList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ASNCountsList) Less(i, j int) bool { return p[i].Value < p[j].Value }