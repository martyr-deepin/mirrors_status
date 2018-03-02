package main

import (
	"flag"
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

func main() {
	var host string
	var user string
	var password string
	var dbname string
	flag.StringVar(&host, "host", "http://influxdb.trend.deepin.io:10086", "influxdb address")
	flag.StringVar(&user, "user", "admin", "influxdb user")
	flag.StringVar(&password, "password", "admin", "influxdb password")
	flag.StringVar(&dbname, "db", "mirror_status", "influxdb database name")
	flag.Parse()

	c, err := NewInfluxClient(host, user, password, dbname)
	if err != nil {
		fmt.Println("E:", err)
		return
	}
	defer c.Close()

	for _, arg := range flag.Args() {
		vs, err := loadOne(arg)
		if err != nil {
			fmt.Println("E:", err)
			continue
		}
		err = PushMirrorStatus(c, vs)
		if err != nil {
			fmt.Println("E:", err)
		}
		fmt.Printf("Pushed %q with %d items\n", arg, len(vs))
	}
}
