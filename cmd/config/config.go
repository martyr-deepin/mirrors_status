package configs

import (
	"io/ioutil"
	"mirrors_status/cmd/log"

	"gopkg.in/yaml.v3"
)

type InfluxDBConf struct {
	Host     string `yaml:"host"`
	Port     string    `yaml:"port"`
	DBName   string `yaml:"dbName"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type HttpConf struct {
	Port string `yaml:"port"`
}

type ServerConf struct {
	InfluxDB *InfluxDBConf `yaml:"influxdb"`
	Http     *HttpConf     `yaml:"http"`
}

func (c *ServerConf) ErrHandler(op string, err error) {
	if err != nil {
		log.Fatalf("%s found error: %v", op, err)
	}
}

func (c *ServerConf) GetConfig() *ServerConf {
	log.Info("Loading configs")
	ymlFile, err := ioutil.ReadFile("configs/config.yml")
	c.ErrHandler("openning file", err)

	err = yaml.Unmarshal(ymlFile, c)
	c.ErrHandler("unmarshal yaml", err)

	return c
}
