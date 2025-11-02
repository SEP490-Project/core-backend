package iproxies

import "context"

type PayOSProxy interface {
	GeneratePaymentLink(ctx context.Context) error
}
