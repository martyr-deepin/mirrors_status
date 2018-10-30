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

func pushToMirrors(c *InfluxClient, points []mirrorsPoint, t time.Time) error {
	var cPoints []*client.Point
	for _, p := range points {
		point, err := client.NewPoint(
			"mirrors",
			map[string]string{
				"name": p.Name,
			},
			map[string]interface{}{
				"progress": p.Progress,
				"latency":  0,
			},
			t)
		if err != nil {
			panic(err)
		}

		cPoints = append(cPoints, point)
	}
	return c.Write(cPoints...)
}

func pushToMirrorsCdn(c *InfluxClient, points []mirrorsCdnPoint, t time.Time) error {
	var cPoints []*client.Point
	for _, p := range points {
		point, err := client.NewPoint(
			"mirrors_cdn",
			map[string]string{
				"mirror_id":    p.MirrorId,
				"node_ip_addr": p.NodeIpAddr,
			},
			map[string]interface{}{
				"progress": p.Progress,
			},
			t)
		if err != nil {
			panic(err)
		}

		cPoints = append(cPoints, point)
	}
	return c.Write(cPoints...)
}

type mirrorsPoint struct {
	Name     string
	Progress float64
}

type mirrorsCdnPoint struct {
	MirrorId   string
	NodeIpAddr string
	Progress   float64
}
