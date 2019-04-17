package mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"mirrors_status/pkg/log"
	"strconv"
)

type Client struct {
	Username string
	Password string
	Host     string
	Port     int
	DbName   string

	DB *gorm.DB
}

func (c *Client) NewMySQLClient() (err error) {
	dbUrl := c.Username + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.Itoa(c.Port) + ")/" + c.DbName + "?charset=utf8&parseTime=True&loc=Local"
	c.DB, err = gorm.Open("mysql", dbUrl)
	if err != nil {
		log.Errorf("Init MySQL client found error:%v", err)
	}
	//defer c.DB.Close()
	return
}
