package queue

import (
	"core-backend/config"

	"github.com/hibiken/asynq"
)

type AsynqClient struct {
	Client *asynq.Client
}

type AsynqServer struct {
	Server *asynq.Server
}

func NewAsynqClient() *AsynqClient {
	cfg := config.GetAppConfig().Asynq
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		DB:       cfg.RedisDB,
		Password: cfg.RedisPassword,
	})
	return &AsynqClient{Client: client}
}

func NewAsynqServer() *AsynqServer {
	cfg := config.GetAppConfig().Asynq
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			DB:       cfg.RedisDB,
			Password: cfg.RedisPassword,
		},
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues:      cfg.Queues,
		},
	)
	return &AsynqServer{Server: server}
}
