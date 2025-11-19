package consumer

import (
	"context"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

type VideoUploadConsumer struct {
	appRegistry        *application.ApplicationRegistry
	s3StreamingStorage irepository_third_party.S3StreamingStorage
	fileRepository     irepository.GenericRepository[model.File]
	unitOfWork         irepository.UnitOfWork
}

func NewVideoUploadConsumer(appRegistry *application.ApplicationRegistry) *VideoUploadConsumer {
	return &VideoUploadConsumer{
		appRegistry:        appRegistry,
		s3StreamingStorage: appRegistry.InfrastructureRegistry.ThirdPartyStorage.S3StreamStorage,
		fileRepository:     appRegistry.DatabaseRegistry.FileRepository,
		unitOfWork:         appRegistry.InfrastructureRegistry.UnitOfWork,
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

	// Get file record from DB
	fileRecord, err := c.fileRepository.GetByID(ctx, msg.FileID, nil)
	if err != nil {
		zap.L().Error("❌ Failed to get file record from DB",
			zap.String("fileID", msg.FileID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get file record: %w", err)
	}

	// Open temp file
	file, err := os.Open(msg.FilePath)
	if err != nil {
		zap.L().Error("❌ Failed to open temp file", zap.String("path", msg.FilePath), zap.Error(err))
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Start transations
	uow := c.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			c.s3StreamingStorage.Delete(ctx, msg.Key)
		}
	}()

	// Upload to S3
	if err := c.s3StreamingStorage.Put(ctx, msg.Key, file, "video/mp4"); err != nil {
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

	// cdnVideoURL := c.s3StreamingStorage.BuildUrl(msg.Key)
	fileRecord.URL = c.s3StreamingStorage.BuildUrl(msg.Key)
	fileRecord.Status = enum.FileStatusUploaded
	fileRecord.UploadedAt = utils.PtrOrNil(time.Now())
	if err := c.fileRepository.Update(ctx, fileRecord); err != nil {
		zap.L().Error("❌ Failed to update file record",
			zap.String("fileID", msg.FileID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update file record: %w", err)
	}

	if err := uow.Commit(); err != nil {
		zap.L().Error("❌ Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
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
