package influxdb

import (
	"fmt"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/model/mirror"
	"strconv"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

func write(ps ...*client.Point) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: configs.NewServerConfig().InfluxDB.DBName,
	})
	if err != nil {
		return err
	}
	for _, p := range ps {
		bp.AddPoint(p)
	}
	return clt.Write(bp)
}

var clt client.Client

func InitInfluxClient() {
	c := configs.NewServerConfig().InfluxDB
	host := c.Host
	port := c.Port
	addr := "http://" + host + ":" + strconv.Itoa(port)
	dbName := c.DBName
	username := c.Username
	password := c.Password
	clt, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	})
	if err != nil {
		panic(err)
	}
	_, _, err = clt.Ping(time.Second)
	if err != nil {
		panic(err)
	}
	_, err = clt.Query(client.Query{
		Command: fmt.Sprintf("create database %s", dbName),
	})
	return
}

func NewInfluxClient() (ct client.Client) {
	return clt
}

func QueryDB(cmd string) (res []client.Result, err error) {
	log.Infof("Query influxdb:%s", cmd)
	q := client.Query{
		Command: cmd,
		Database: configs.NewServerConfig().InfluxDB.DBName,
	}
	if resp, e := clt.Query(q); e == nil {
		if resp.Error() != nil {
			return res, resp.Error()
		}
		res = resp.Results
	} else {
		return res, err
	}
	return res, nil
}

func PushMirror(t time.Time, point mirror.MirrorsPoint) error {
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
	return write(cPoints...)
}

func PushMirrors(t time.Time, points []mirror.MirrorsPoint) error {
	for _, p := range points {
		err := PushMirror(time.Now(), p)
		if err != nil {
			return err
		}
	}
	return nil
}

func PushMirrorCdn(t time.Time, point mirror.MirrorsCdnPoint) error {
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
	return write(cPoints...)
}

func PushMirrorsCdn(t time.Time, points []mirror.MirrorsCdnPoint) error {
	for _, p := range points {
		err := PushMirrorCdn(time.Now(), p)
		if err != nil {
			return err
		}
	}
	return nil
}


