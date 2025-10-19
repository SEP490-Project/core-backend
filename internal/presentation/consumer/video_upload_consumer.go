package consumer

import (
	"context"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
)

type VideoUploadConsumer struct {
	appRegistry *application.ApplicationRegistry
	unitOfWork  irepository.UnitOfWork
}

func NewVideoUploadConsumer(appRegistry *application.ApplicationRegistry) *VideoUploadConsumer {
	return &VideoUploadConsumer{
		appRegistry: appRegistry,
		unitOfWork:  appRegistry.InfrastructureRegistry.UnitOfWork,
	}
}

// Handle xử lý từng message nhận từ RabbitMQ
func (c *VideoUploadConsumer) Handle(ctx context.Context, body []byte) error {
	var msg consumers.VideoUploadMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("❌ Failed to unmarshal VideoUploadMessage", zap.Error(err))
		return fmt.Errorf("failed to unmarshal VideoUploadMessage: %w", err)
	}

	zap.L().Info("📥 Received video upload task",
		zap.String("userID", msg.UserID),
		zap.String("filePath", msg.FilePath),
		zap.String("key", msg.Key),
		zap.String("action", fmt.Sprintf("%v", msg.Action)),
	)

	// Open temp file
	file, err := os.Open(msg.FilePath)
	if err != nil {
		zap.L().Error("❌ Failed to open temp file", zap.String("path", msg.FilePath), zap.Error(err))
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload to S3
	videoStorage := c.appRegistry.InfrastructureRegistry.ThirdPartyStorage.S3StreamStorage
	if err := videoStorage.Put(ctx, msg.Key, file, "video/mp4"); err != nil {
		zap.L().Error("❌ Failed to upload video to S3",
			zap.String("key", msg.Key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to upload to s3: %w", err)
	}

	// Delete temp file
	if err := os.Remove(msg.FilePath); err != nil {
		zap.L().Warn("⚠️ Failed to remove temp file after upload",
			zap.String("path", msg.FilePath),
			zap.Error(err),
		)
	} else {
		zap.L().Info("🧼 Temp file removed", zap.String("path", msg.FilePath))
	}

	zap.L().Info("✅ Video upload completed successfully",
		zap.String("key", msg.Key),
		zap.String("userID", msg.UserID),
	)

	//update database or perform additional actions if needed
	if msg.Action != nil {
		zap.L().Info("Performing post-upload action", zap.String("action", *msg.Action))
		//do something with msg.Action
	}
	return nil
}
