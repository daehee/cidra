package main

import (
	"encoding/csv"
	"os"
	"sync"
)

func output(wg *sync.WaitGroup, out <-chan []string) {
	w := csv.NewWriter(os.Stdout)

	for o := range out {
		err := w.Write(o)
		check(err)
	}

	w.Flush()
	err := w.Error()
	check(err)

	wg.Done()
}
