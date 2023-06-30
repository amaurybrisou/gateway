package store

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog/log"
)

type Persister interface {
	InsertOne(ctx context.Context, dataBase, col string, doc interface{}) (string, error)
}

type RedisClient struct {
	addr string
	port int
	*redis.Client
}

func NewRedisClient(ctx context.Context, addr string, port int) (*RedisClient, error) {
	r := &RedisClient{addr: addr, port: port}
	err := r.start(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("redis client")
		return nil, err
	}

	return r, nil
}

func (r *RedisClient) start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", r.addr, r.port)
	conn := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	log.Ctx(ctx).Info().Str("address", addr).Msg("start redis client")

	_, err := conn.Ping().Result()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("redis client")
		return err
	}

	r.Client = conn

	return nil
}

func (r *RedisClient) Close(ctx context.Context) error {
	if r.Client == nil {
		return nil
	}
	log.Ctx(ctx).Info().Str("address", r.Client.String()).Msg("closing redis client")
	return r.Client.Close()
}

func (r *RedisClient) InsertOne(ctx context.Context, dataBase, col string, doc interface{}) (string, error) {
	return "", nil
}
