// Package proxies contain the proxies implementations and a singleton http client
package proxies

import (
	"core-backend/config"
	"core-backend/internal/application/interfaces/iproxies"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type ProxiesRegistry struct {
	httpClient      *http.Client
	PayOSProxy      iproxies.PayOSProxy
	GHNProxy        iproxies.GHNProxy
	FacebookProxy   iproxies.FacebookProxy
	TikTokProxy     iproxies.TikTokProxy
	AIClientManager iproxies.AIClientManager
	db              *gorm.DB
}

func NewProxiesRegistry(config *config.AppConfig, db *gorm.DB) *ProxiesRegistry {
	transport := &http.Transport{
		MaxIdleConns:          config.HTTPClient.MaxIdleConns,
		MaxIdleConnsPerHost:   config.HTTPClient.MaxIdleConnsPerHost,
		IdleConnTimeout:       time.Duration(config.HTTPClient.IdleConnTimeout) * time.Second,
		TLSHandshakeTimeout:   time.Duration(config.HTTPClient.TLSHandshakeTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(config.HTTPClient.ExpectContinueTimeout) * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.HTTPClient.Timeout) * time.Second,
	}

	return &ProxiesRegistry{
		httpClient:    client,
		PayOSProxy:    NewPayOSProxy(client, config),
		GHNProxy:      NewGHNProxy(client, config, db),
		FacebookProxy: NewFacebookProxy(client, config),
		TikTokProxy:   NewTikTokProxy(client, config),
		//AIClientManager: ai.NewAIClientManager(client, &config.AI),
	}
}

func (reg *ProxiesRegistry) GetHTTPClient() *http.Client {
	return reg.httpClient
}
