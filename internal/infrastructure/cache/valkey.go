package cache

import (
	"context"
	"core-backend/config"

	"github.com/redis/go-redis/v9"
)

type Valkey struct {
	Client *redis.Client
}

func NewValkey() *Valkey {
	cfg := config.GetAppConfig().Cache
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + string(cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Valkey{Client: client}
}

func (v *Valkey) Set(ctx context.Context, key string, value interface{}) error {
	return v.Client.Set(ctx, key, value, 0).Err()
}

func (v *Valkey) Get(ctx context.Context, key string) (string, error) {
	return v.Client.Get(ctx, key).Result()
}

func (v *Valkey) Del(ctx context.Context, key string) error {
	return v.Client.Del(ctx, key).Err()
}
