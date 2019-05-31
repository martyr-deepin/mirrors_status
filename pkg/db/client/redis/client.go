package redis

import (
	"mirrors_status/internal/config"
	"strconv"
	"time"
	"github.com/go-redis/redis"
)

var client *redis.Client

func InitRedisClient() {
	conf := configs.NewServerConfig().Redis
	client = redis.NewClient(&redis.Options{
		Addr: conf.Host + ":" + strconv.Itoa(conf.Port),
		Password: conf.Password,
		DB: conf.DBName,
	})
	if _, err := client.Ping().Result(); err != nil {
		panic(err)
	}
	return
}

func NewRedisClient() (client *redis.Client) {
	return client
}

func Set(key, val string, duration time.Duration) error {
	err := client.Set(key, val, duration).Err()
	if err != nil {
		return err
	}
	return nil
}

func Get(key string) (string, error) {
	val, err := client.Get(key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func Del(keys... string) error {
	return client.Del(keys...).Err()
}