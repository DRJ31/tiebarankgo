package model

import (
	"fmt"
	"github.com/DRJ31/tiebarankgo/config"
	"github.com/redis/go-redis/v9"
)

func InitRedis() *redis.Client {
	cf := config.GetConfig()
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%v:%v", cf.RedisHost, cf.RedisPort),
		Password: cf.RedisPass,
		DB:       0,
	})
}
