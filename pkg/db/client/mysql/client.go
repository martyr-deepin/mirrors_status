package mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"strconv"
)

var client *gorm.DB

func InitMySQLClient() {
	c := configs.NewServerConfig().MySQLDB
	dbUrl := c.Username + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.Itoa(c.Port) + ")/" + c.DBName + "?charset=utf8&parseTime=True&loc=Local"
	clt, err := gorm.Open("mysql", dbUrl)
	if err != nil {
		log.Errorf("Init MySQL client found error:%v", err)
		panic(err)
	}
	client = clt.New()
	return
}

func NewMySQLClient() (db *gorm.DB) {
	return client
}