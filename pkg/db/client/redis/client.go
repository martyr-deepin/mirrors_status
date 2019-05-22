package redis

import (
	"strconv"
	"time"
	"github.com/go-redis/redis"
)

type Client struct {
	Host string
	Port int
	Username string
	Password string
	DBName int

	c *redis.Client
}

func (c *Client) NewRedisClient() (err error) {
	c.c = redis.NewClient(&redis.Options{
		Addr: c.Host + ":" + strconv.Itoa(c.Port),
		Password: c.Password,
		DB: c.DBName,
	})
	if _, err := c.c.Ping().Result(); err != nil {
		return err
	}
	return nil
}

func (c *Client) Set(key, val string, duration time.Duration) error {
	err := c.c.Set(key, val, duration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Get(key string) (string, error) {
	val, err := c.c.Get(key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *Client) Del(keys... string) error {
	return c.c.Del(keys...).Err()
}