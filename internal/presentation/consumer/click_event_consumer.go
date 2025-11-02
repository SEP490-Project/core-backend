package consumer

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"encoding/json"
	"errors"

	"go.uber.org/zap"
)

type ClickEventConsumer struct {
	clickEventRepo irepository.ClickEventRepository
}

func NewClickEventConsumer(clickEventRepo irepository.ClickEventRepository) *ClickEventConsumer {
	return &ClickEventConsumer{
		clickEventRepo: clickEventRepo,
	}
}

// Handle processes click event messages from RabbitMQ and persists them to the click_events hypertable
func (c *ClickEventConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Debug("Processing click event message")

	// Unmarshal message
	var message consumers.ClickEventMessage
	if err := json.Unmarshal(body, &message); err != nil {
		zap.L().Error("Failed to unmarshal click event message",
			zap.Error(err),
			zap.String("body", string(body)))
		return err
	}

	// Validate required fields
	if err := c.validateMessage(&message); err != nil {
		zap.L().Error("Invalid click event message", zap.Error(err))
		return err
	}

	// Skip bot clicks (optional - can be configured)
	if message.IsBot {
		zap.L().Debug("Skipping bot click event",
			zap.String("affiliate_link_id", message.AffiliateLinkID.String()),
			zap.String("user_agent", message.UserAgent))
		return nil // Don't requeue, just skip
	}

	// Create click event entity
	clickEvent := &model.ClickEvent{
		AffiliateLinkID: message.AffiliateLinkID,
		UserID:          message.UserID,
		ClickedAt:       message.ClickedAt,
		IPAddress:       &message.IPAddress,
		UserAgent:       &message.UserAgent,
		ReferrerURL:     message.ReferrerURL,
		SessionID:       message.SessionID,
	}

	// Persist to click_events hypertable (TimescaleDB)
	if err := c.clickEventRepo.Add(ctx, clickEvent); err != nil {
		zap.L().Error("Failed to persist click event",
			zap.String("affiliate_link_id", message.AffiliateLinkID.String()),
			zap.String("ip_address", message.IPAddress),
			zap.Error(err))
		return err // Return error to trigger RabbitMQ retry
	}

	zap.L().Info("Click event persisted successfully",
		zap.String("click_event_id", clickEvent.ID.String()),
		zap.String("affiliate_link_id", message.AffiliateLinkID.String()),
		zap.String("ip_address", message.IPAddress),
		zap.Bool("authenticated", message.UserID != nil))

	return nil
}

// validateMessage performs basic validation on the click event message
func (c *ClickEventConsumer) validateMessage(message *consumers.ClickEventMessage) error {
	if message.AffiliateLinkID.String() == "" {
		return errors.New("affiliate_link_id is required")
	}

	if message.IPAddress == "" {
		return errors.New("ip_address is required")
	}

	if message.UserAgent == "" {
		return errors.New("user_agent is required")
	}

	if message.ClickedAt.IsZero() {
		return errors.New("clicked_at timestamp is required")
	}

	return nil
}
