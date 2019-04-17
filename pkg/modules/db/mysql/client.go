package mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"mirrors_status/pkg/log"
<<<<<<< HEAD
=======
	"strconv"
>>>>>>> zhaojuwen/sync-check
)

type Client struct {
	Username string
	Password string
	Host     string
<<<<<<< HEAD
	Port     string
=======
	Port     int
>>>>>>> zhaojuwen/sync-check
	DbName   string

	DB *gorm.DB
}

func (c *Client) NewMySQLClient() (err error) {
<<<<<<< HEAD
	dbUrl := c.Username + ":" + c.Password + "@tcp(" + c.Host + ":" + c.Port + ")/" + c.DbName + "?charset=utf8&parseTime=True&loc=Local"
=======
	dbUrl := c.Username + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.Itoa(c.Port) + ")/" + c.DbName + "?charset=utf8&parseTime=True&loc=Local"
>>>>>>> zhaojuwen/sync-check
	c.DB, err = gorm.Open("mysql", dbUrl)
	if err != nil {
		log.Errorf("Init MySQL client found error:%v", err)
	}
	//defer c.DB.Close()
	return
<<<<<<< HEAD
}
=======
}

>>>>>>> zhaojuwen/sync-check
