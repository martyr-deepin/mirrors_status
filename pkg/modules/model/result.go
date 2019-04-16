package model

import (
	"fmt"
	"time"
)

type OldResult struct {
	Name        string
	Support2014 bool
	Support2015 bool

	LastSync time.Time

	Latency   int64
	Progress  float64
	CheckTime time.Time
}

func Show(vs []OldResult) {
	var cache = make(map[string][]OldResult)
	for _, v := range vs {
		cache[v.Name] = append(cache[v.Name], v)
	}
	for n, v := range cache {
		fmt.Printf("%s has %d data\n", n, len(v))
	}
}

