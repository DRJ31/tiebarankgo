package model

import (
	"fmt"
	"github.com/DRJ31/tiebarankgo/config"
	"github.com/go-redis/redis/v8"
)

func InitRedis() *redis.Client {
	cf := config.GetConfig()
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%v:%v", cf.RedisHost, cf.RedisPort),
		Password: "",
		DB:       0,
	})
}
