package influxdb

import (
	"fmt"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

func Write(ps ...*client.Point) error {
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
	log.Infof("trying connecting influxdb:%s %s", addr, username)
	var err error
	clt, err = client.NewHTTPClient(client.HTTPConfig{
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
}

func NewInfluxClient() (ct client.Client) {
	return clt
}

func queryDB(cmd string) (res []client.Result, err error) {
	log.Infof("Query influxdb:%s", cmd)
	q := client.Query{
		Command: cmd,
		Database: configs.NewServerConfig().InfluxDB.DBName,
	}
	if resp, e := NewInfluxClient().Query(q); e == nil {
		if resp.Error() != nil {
			return res, resp.Error()
		}
		res = resp.Results
	} else {
		return res, err
	}
	return res, nil
}

func Select(measurement string, tags []string, where map[string]interface{}) (
	data [][]interface{}, err error) {
	clause := make([]string, 0)
	for tag, val := range where {
		typ := reflect.TypeOf(val)
		switch typ.Kind() {
		case reflect.Int, reflect.Int64:
			clause = append(clause, fmt.Sprintf(`"%s" = %d`, tag, val))
		case reflect.String:
			clause = append(clause, fmt.Sprintf(`"%s" = '%s'`, tag, val))
		}
	}
	query := fmt.Sprintf(`select %s from %s`, strings.Join(tags, ","), measurement)
	if len(clause) != 0 {
		query += " where " + strings.Join(clause, " and ")
	}
	rawdata, err := queryDB(query)
	if err != nil {
		return
	}
	if len(rawdata) == 0 || len(rawdata[0].Series) == 0 {
		err = fmt.Errorf("influxdb return empty value")
		return
	}
	data = rawdata[0].Series[0].Values
	return
}

func LastValue(measurement, tag, field string) (data map[string]Value, err error) {
	query := fmt.Sprintf(`select last(%s) from %s group by "%s"`, field, measurement, tag)
	rawdata, err := queryDB(query)
	if err != nil {
		return
	}
	if len(rawdata) == 0 {
		err = fmt.Errorf("influxdb return empty value")
		return
	}
	results := rawdata[0]
	data = make(map[string]Value, len(results.Series))
	for _, serial := range results.Series {
		if serial.Values == nil || len(serial.Values) == 0 {
			log.Info("influxdb returned 0 value")
			continue
		} else {
			values := serial.Values[0]
			if len(values) < 2 {
				log.Info("influxdb return invalid value")
				continue
			}
			timestamp, _ := time.Parse(time.RFC3339Nano, values[0].(string))
			value := values[1]
			data[serial.Name] = Value{
				Value:     value.(float64),
				Timestamp: timestamp,
			}
			log.Infof("mirror: %v, progress: %v", serial.Name, value.(float64))
		}
	}
	return
}

func LatestMirrorData(measurement, last, tag string, where map[string]interface{}, group string) (data [][]interface{}, err error) {
	clause := make([]string, 0)
	for t, val := range where {
		typ := reflect.TypeOf(val)
		switch typ.Kind() {
		case reflect.Int, reflect.Int64:
			clause = append(clause, fmt.Sprintf(`"%s" = %d`, t, val))
		case reflect.String:
			clause = append(clause, fmt.Sprintf(`"%s" = '%s'`, t, val))
		}
	}
	query := fmt.Sprintf(`select last(%s)`, last)
	if tag != "" {
		query += fmt.Sprintf(",%s", tag)
	}
	query += fmt.Sprintf(" from %s", measurement)
	if len(clause) != 0 {
		query += " where " + strings.Join(clause, " and ")
	}
	if group != "" {
		query += fmt.Sprintf(" group by %s", group)
	}
	rawdata, err := queryDB(query)
	if err != nil {
		return
	}
	if len(rawdata) == 0 || len(rawdata[0].Series) == 0 {
		err = fmt.Errorf("influxdb return empty value")
		return
	}
	data = rawdata[0].Series[0].Values
	return
}

func LatestCdnData(measurement, last, tag string, where map[string]interface{}, group string) (data [][][]interface{}, err error) {
	clause := make([]string, 0)
	for t, val := range where {
		typ := reflect.TypeOf(val)
		switch typ.Kind() {
		case reflect.Int, reflect.Int64:
			clause = append(clause, fmt.Sprintf(`"%s" = %d`, t, val))
		case reflect.String:
			clause = append(clause, fmt.Sprintf(`"%s" = '%s'`, t, val))
		}
	}
	query := fmt.Sprintf(`select last(%s)`, last)
	if tag != "" {
		query += fmt.Sprintf(",%s", tag)
	}
	query += fmt.Sprintf(" from %s", measurement)
	if len(clause) != 0 {
		query += " where " + strings.Join(clause, " and ")
	}
	if group != "" {
		query += fmt.Sprintf(" group by %s", group)
	}
	rawdata, err := queryDB(query)
	if err != nil {
		return
	}
	if len(rawdata) == 0 || len(rawdata[0].Series) == 0 {
		err = fmt.Errorf("influxdb return empty value")
		return
	}
	for _, d := range rawdata {
		for _, s := range d.Series {
			data = append(data, s.Values)
		}
	}
	//data = rawdata[0].Series[0].Values
	return
}

type Data struct {
	Results []struct {
		Series []struct {
			Tags struct {
				Name string `json:"name"`
			} `json:"tags"`
			Values [][]interface{} `json:"values"`
		} `json:"series"`
	} `json:"results"`
}

type Value struct {
	Value     interface{}
	Timestamp time.Time
}