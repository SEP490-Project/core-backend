package proxies

import (
	"context"
	"core-backend/internal/application/interfaces/iproxies"
	"net/http"
)

type payosProxy struct {
	*BaseProxy
}

// GeneratePaymentLink implements iproxies.PayOSProxy.
func (p *payosProxy) GeneratePaymentLink(ctx context.Context) error {
	p.Get(ctx, "/example-path", nil)
	panic("unimplemented")
}

func NewPayOSProxy(httpClient *http.Client, baseURL string) iproxies.PayOSProxy {
	return &payosProxy{&BaseProxy{httpClient: httpClient, baseURL: baseURL}}
}
