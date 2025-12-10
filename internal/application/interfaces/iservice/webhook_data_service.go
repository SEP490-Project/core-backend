package iservice

import (
	"context"
	"core-backend/internal/domain/model"
)

type WebhookDataService interface {
	SaveWebhookData(ctx context.Context, async bool, webhookData *model.WebhookData) error
}
