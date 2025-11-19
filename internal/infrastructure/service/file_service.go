// Package service
package service

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/internal/infrastructure/third_party_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type fileService struct {
	imageStorage irepository_third_party.S3Storage
	videoStorage irepository_third_party.S3StreamingStorage
	fileRepo     irepository.GenericRepository[model.File]
	rabbitmq     *rabbitmq.RabbitMQ
}

// Video stream upload and delete
func (s *fileService) UploadVideoStream(ctx context.Context, req *requests.UploadVideoChunkRequest, data *[]byte) (*responses.PathResponse, error) {
	zap.L().Info("Received video chunk",
		zap.Any("request", req))

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid userID: %w", err)
	}

	// Create or update file record
	fileRecord, err := s.getOrCreateFileRecord(ctx, userUUID, req.FileName, "video")
	if err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// Update status to UPLOADING if not already
	if fileRecord.Status == enum.FileStatusPending {
		fileRecord.Status = enum.FileStatusUploading
		if err = s.fileRepo.Update(ctx, fileRecord); err != nil {
			zap.L().Warn("Failed to update file status to UPLOADING", zap.Error(err))
		}
	}

	tmpDir := s.getTempDir(req.UserID)
	if err = os.MkdirAll(tmpDir, 0o755); err != nil {
		s.markFileFailed(ctx, fileRecord.ID, fmt.Sprintf("failed to create temp dir: %v", err))
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	partPath := s.getTempFilePath(req.UserID, req.FileName)

	// append next chunk
	if err = s.appendChunkToFile(partPath, data); err != nil {
		s.markFileFailed(ctx, fileRecord.ID, fmt.Sprintf("failed to append chunk: %v", err))
		return nil, fmt.Errorf("failed to append chunk: %w", err)
	}

	zap.L().Info("Chunk appended successfully",
		zap.String("userID", req.UserID),
		zap.String("fileName", req.FileName),
	)

	if !req.IsLastChunk {
		return nil, nil
	}

	// Last chunk? assembled file -> upload
	zap.L().Info("Final chunk received, starting upload...",
		zap.String("filePath", partPath),
	)

	finalPath := strings.TrimSuffix(partPath, ".part")
	if err = os.Rename(partPath, finalPath); err != nil {
		s.markFileFailed(ctx, fileRecord.ID, fmt.Sprintf("failed to rename assembled file: %v", err))
		return nil, fmt.Errorf("failed to finalize file (rename): %w", err)
	}

	pathResp, err := s.enqueueVideoUploadTask(ctx, req, finalPath, fileRecord.ID)
	if err != nil {
		s.markFileFailed(ctx, fileRecord.ID, fmt.Sprintf("failed to enqueue video upload task: %v", err))
		return nil, fmt.Errorf("failed to enqueue video upload task: %w", err)
	}

	return &pathResp, nil
}

func (s *fileService) DeleteVideoStream(ctx context.Context, userID string, fileName string) error {
	// remove any temp part file
	partPath := filepath.Join(os.TempDir(), "video_uploads", userID, fileName+".part")
	_ = os.Remove(partPath)

	key := fmt.Sprintf("%s/%s", userID, filepath.Base(fileName))
	if err := s.videoStorage.Delete(context.TODO(), key); err != nil {
		return fmt.Errorf("failed to delete video from streaming repo: %w", err)
	}

	fileRecord, err := s.findFileByKey(ctx, key)
	if err == nil && fileRecord != nil {
		if err := s.fileRepo.DeleteByID(ctx, fileRecord.ID); err != nil {
			zap.L().Warn("Failed to delete file record", zap.Error(err))
		}
	}

	return nil
}

// File upload and delete
func (s *fileService) UploadFile(ctx context.Context, userID string, filePath string, destination string) (string, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid userID: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// ensure filename is clean
	fileName := filepath.Base(destination)
	key := fmt.Sprintf("%s/%s", userID, fileName)

	// Create file record before upload
	fileRecord := &model.File{
		Name:       fileName,
		StorageKey: key,
		MimeType:   "application/octet-stream",
		Size:       fileInfo.Size(),
		Status:     enum.FileStatusUploading,
		UploadedBy: &userUUID,
	}

	if err = s.fileRepo.Add(ctx, fileRecord); err != nil {
		return "", fmt.Errorf("failed to create file record: %w", err)
	}

	// Upload to S3
	err = s.imageStorage.Put(ctx, key, file, "application/octet-stream")
	if err != nil {
		s.markFileFailed(ctx, fileRecord.ID, fmt.Sprintf("failed to upload file: %v", err))
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Build object URL
	url := s.imageStorage.BuildUrl(key)

	// Update file record with URL and UPLOADED status
	fileRecord.URL = url
	fileRecord.Status = enum.FileStatusUploaded
	fileRecord.UploadedAt = utils.PtrOrNil(time.Now())
	if err := s.fileRepo.Update(ctx, fileRecord); err != nil {
		zap.L().Warn("Failed to update file record after upload", zap.Error(err))
	}

	return url, nil
}

func (s *fileService) DeleteFile(ctx context.Context, userID string, fileName string) error {
	key := fmt.Sprintf("%s/%s", userID, filepath.Base(fileName))

	if err := s.imageStorage.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	// Soft delete file record
	fileRecord, err := s.findFileByKey(ctx, key)
	if err == nil && fileRecord != nil {
		if err := s.fileRepo.DeleteByID(ctx, fileRecord.ID); err != nil {
			zap.L().Warn("Failed to delete file record", zap.Error(err))
		}
	}

	return nil
}

func NewFileService(storage3rd *third_party_repository.ThirdPartyStorageRegistry, fileRepo irepository.GenericRepository[model.File], rabbitmq *rabbitmq.RabbitMQ) iservice.FileService {
	return &fileService{
		imageStorage: storage3rd.S3Storage,
		videoStorage: storage3rd.S3StreamStorage,
		fileRepo:     fileRepo,
		rabbitmq:     rabbitmq,
	}
}

// region: ============ Helper Methods =============

func (s *fileService) getTempDir(userID string) string {
	//return filepath.Join(os.TempDir(), userID) //prod
	return filepath.Join("tmp", userID) //test
}

func (s *fileService) getTempFilePath(userID string, fileName string) string {
	return filepath.Join(s.getTempDir(userID), fileName+".part")
}

func (s *fileService) appendChunkToFile(path string, data *[]byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			zap.L().Error("Failed to close file", zap.Error(err))
		}
	}(f)

	if _, err := f.Write(*data); err != nil {
		return err
	}
	return nil
}

func (s *fileService) enqueueVideoUploadTask(ctx context.Context, req *requests.UploadVideoChunkRequest, finalPath string, fileID uuid.UUID) (responses.PathResponse, error) {
	// RabbitMQ handle this step
	key := fmt.Sprintf("%s/%s", req.UserID, filepath.Base(req.FileName))
	url := s.videoStorage.BuildUrl(key)

	pathResp := responses.PathResponse{
		HostURL: url,
		TempURL: url,
	}

	videoMessage := &consumers.VideoUploadMessage{
		UserID:          req.UserID,
		FilePath:        finalPath,
		Key:             key,
		Action:          nil,
		FileID:          fileID.String(),
		IsHLS:           req.IsHLS,
		Resolutions:     req.GetResolutions(),
		SegmentDuration: req.SegmentDuration,
	}
	// build payload
	payload, err := json.Marshal(videoMessage)
	if err != nil {
		zap.L().Error("Failed to marshal video upload message", zap.Error(err))
		return pathResp, fmt.Errorf("failed to marshal video upload message: %w", err)
	}

	// get producer from producer manager
	videoProducer, err := s.rabbitmq.GetProducer("video-upload-producer")
	if err != nil {
		zap.L().Error("Failed to get video-upload-producer", zap.Error(err))
		return pathResp, fmt.Errorf("failed to get video-upload-producer: %w", err)
	}

	// publish message to RabbitMQ
	if err := videoProducer.Publish(ctx, payload); err != nil {
		zap.L().Error("Failed to publish video upload task", zap.Error(err))
		return pathResp, fmt.Errorf("failed to publish video upload task: %w", err)
	}

	zap.L().Info("✅ Video upload task enqueued",
		zap.String("userID", req.UserID),
		zap.String("fileName", req.FileName),
		zap.String("finalPath", finalPath),
		zap.String("s3Key", key),
		zap.String("fileID", fileID.String()),
	)

	return pathResp, nil
}

func (s *fileService) getOrCreateFileRecord(ctx context.Context, userID uuid.UUID, fileName string, fileType string) (*model.File, error) {
	key := fmt.Sprintf("%s/%s", userID.String(), filepath.Base(fileName))

	// Try to find existing record
	existingFile, err := s.findFileByKey(ctx, key)
	if err == nil && existingFile != nil {
		return existingFile, nil
	}

	// Create new record
	mimeType := "video/mp4"
	if fileType == "image" {
		mimeType = "image/jpeg"
	}

	fileRecord := &model.File{
		Name:       fileName,
		StorageKey: key,
		MimeType:   mimeType,
		Status:     enum.FileStatusPending,
		UploadedBy: &userID,
	}

	if err := s.fileRepo.Add(ctx, fileRecord); err != nil {
		return nil, err
	}

	return fileRecord, nil
}

func (s *fileService) findFileByKey(ctx context.Context, key string) (*model.File, error) {
	file, err := s.fileRepo.GetByCondition(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("storage_key = ?", key)
		},
		nil)
	return file, err
}

func (s *fileService) markFileFailed(ctx context.Context, fileID uuid.UUID, reason string) {
	file, err := s.fileRepo.GetByID(ctx, fileID, nil)
	if err != nil {
		zap.L().Warn("Failed to get file for status update", zap.Error(err))
		return
	}

	file.Status = enum.FileStatusFailed
	file.ErrorReason = &reason
	if err := s.fileRepo.Update(ctx, file); err != nil {
		zap.L().Warn("Failed to mark file as failed", zap.Error(err))
	}
}

// endregion
