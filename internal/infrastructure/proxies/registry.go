// Package proxies contain the proxies implementations and a singleton http client
package proxies

import (
	"core-backend/config"
	"core-backend/internal/application/interfaces/iproxies"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type ProxiesRegistry struct {
	httpClient *http.Client
	PayOSProxy iproxies.PayOSProxy
	GHNProxy   iproxies.GHNProxy
	db         *gorm.DB
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
		httpClient: client,
		PayOSProxy: NewPayOSProxy(
			client,
			config.PayOS.BaseURL,
			config.PayOS.ClientID,
			config.PayOS.APIKey,
			config.PayOS.ChecksumKey,
		),
		GHNProxy: NewGHNProxy(
			client,
			config,
			db,
		),
	}
}

func (reg *ProxiesRegistry) GetHTTPClient() *http.Client {
	return reg.httpClient
}
