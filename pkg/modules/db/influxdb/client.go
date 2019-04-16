package influxdb

import (
	"fmt"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/model"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

type Client struct {
	Username string
	Password string
	Host     string
	Port     string
	DbName   string

	c client.Client
}

func (c *Client) write(ps ...*client.Point) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: c.DbName,
	})
	if err != nil {
		return err
	}
	for _, p := range ps {
		bp.AddPoint(p)
	}
	return c.c.Write(bp)
}

func (c *Client) close() error { return c.c.Close() }

func (c *Client) NewInfluxClient() (err error) {
	host := c.Host
	port := c.Port
	addr := "http://" + host + ":" + port
	dbName := c.DbName
	username := c.Username
	password := c.Password
	c.c, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	})
	if err != nil {
		return err
	}
	_, _, err = c.c.Ping(time.Second)
	if err != nil {
		return err
	}
	_, err = c.c.Query(client.Query{
		Command: fmt.Sprintf("create database %s", dbName),
	})
	defer c.c.Close()
	return
}

func (c * Client) QueryDB(cmd string) (res []client.Result, err error) {
	log.Infof("Query influxdb:%s", cmd)
	q := client.Query{
		Command: cmd,
		Database: c.DbName,
	}
	if resp, e := c.c.Query(q); e == nil {
		if resp.Error() != nil {
			return res, resp.Error()
		}
		res = resp.Results
	} else {
		return res, err
	}
	return res, nil
}

func (c *Client) PushMirror(t time.Time, point model.MirrorsPoint) error {
	var cPoints []*client.Point
	p, err := client.NewPoint(
		"mirrors",
		map[string]string{
			"name": point.Name,
		},
		map[string]interface{} {
			"progress": point.Progress,
			"latency": 0,
		},
		t)
	if err != nil {
		return err
	}
	log.Infof("Pushing mirror:%v", err)
	cPoints = append(cPoints, p)
	return c.write(cPoints...)
}

func (c *Client) PushMirrors(t time.Time, points []model.MirrorsPoint) error {
	for _, p := range points {
		err := c.PushMirror(time.Now(), p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) PushMirrorCdn(t time.Time, point model.MirrorsCdnPoint) error {
	var cPoints []*client.Point
	p, err := client.NewPoint(
		"mirrors_cdn",
		map[string]string{
			"mirror_id":    point.MirrorId,
			"node_ip_addr": point.NodeIpAddr,
		},
		map[string]interface{}{
			"progress": point.Progress,
		},
		t)
	if err != nil {
		return err
	}
	cPoints = append(cPoints, p)
	return c.write(cPoints...)
}

func (c *Client) PushMirrorsCdn(t time.Time, points []model.MirrorsCdnPoint) error {
	for _, p := range points {
		err := c.PushMirrorCdn(time.Now(), p)
		if err != nil {
			return err
		}
	}
	return nil
}


