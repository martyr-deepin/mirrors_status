package main

import (
	"fmt"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

type InfluxClient struct {
	c      client.Client
	dbname string
}

func (c *InfluxClient) Write(ps ...*client.Point) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: c.dbname,
	})
	if err != nil {
		return err
	}
	for _, p := range ps {
		bp.AddPoint(p)
	}
	return c.c.Write(bp)
}

func (c *InfluxClient) Close() error { return c.c.Close() }

func NewInfluxClient(addr string, user string, passwd string, dbname string) (*InfluxClient, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: user,
		Password: passwd,
	})
	if err != nil {
		return nil, err
	}
	_, _, err = c.Ping(time.Second)
	if err != nil {
		return nil, err
	}
	_, err = c.Query(client.Query{
		Command: fmt.Sprintf("create database %s", dbname),
	})
	return &InfluxClient{c, dbname}, err
}

func pushMirrorStatus(c *InfluxClient, vs []dbTestResultItem, ts time.Time) error {
	var data []*client.Point
	for _, v := range vs {
		p, err := client.NewPoint(
			"mirrors",
			map[string]string{
				"name": v.Name,
			},
			map[string]interface{}{
				"progress": v.Progress,
				"latency":  0,
			},
			ts)
		if err != nil {
			panic(err)
		}

		data = append(data, p)
	}
	return c.Write(data...)
}

type dbTestResultItem struct {
	Name     string
	Progress float64
}
