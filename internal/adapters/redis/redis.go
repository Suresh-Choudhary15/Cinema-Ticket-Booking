package redis

import (
	"context"
	"log"

	goredis "github.com/redis/go-redis/v9"
)

func NewClient(addr string) *goredis.Client {
	opts, err := goredis.ParseURL(addr)
	if err != nil {
		// fall back to plain host:port format for local dev
		opts = &goredis.Options{Addr: addr}
	}

	rdb := goredis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}
	log.Printf("connected to redis")

	return rdb
}
