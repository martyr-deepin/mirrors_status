package main

import (
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"io"
	"time"
)

type InfluxClient struct {
	c      client.Client
	dbname string
}
type DumpClient struct {
	w io.Writer
}

func (c DumpClient) Write(ps ...*client.Point) error {
	for _, p := range ps {
		fmt.Println(c.w, p.String())
	}
	return nil
}
func (c DumpClient) Close() error { return nil }

type DataSource interface {
	Write(ps ...*client.Point) error
	Close() error
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
