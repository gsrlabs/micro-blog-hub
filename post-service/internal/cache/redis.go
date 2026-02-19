package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRedisClient(parent context.Context, logger *zap.Logger, host, port string) (*redis.Client, error) {

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
    defer cancel()
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil

}
