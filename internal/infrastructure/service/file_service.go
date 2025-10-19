// Package service
package service

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/internal/infrastructure/third_party_repository"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type fileService struct {
	imageStorage irepository_third_party.S3Storage
	videoStorage irepository_third_party.S3StreamingStorage
	rabbitmq     *rabbitmq.RabbitMQ
}

// Video stream upload and delete
func (s *fileService) UploadVideoStream(userID string, fileName string, data []byte, isLastChunk bool, action *string) (*responses.PathResponse, error) {
	tmpDir := s.getTempDir(userID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	partPath := s.getTempFilePath(userID, fileName)

	// append next chunk
	if err := s.appendChunkToFile(partPath, data); err != nil {
		return nil, fmt.Errorf("failed to append chunk: %w", err)
	}

	zap.L().Info("Chunk appended successfully",
		zap.String("userID", userID),
		zap.String("fileName", fileName),
		zap.Int("chunkSize", len(data)),
	)

	if !isLastChunk {
		return nil, nil
	}

	// Last chunk? assembled file -> upload
	zap.L().Info("Final chunk received, starting upload...",
		zap.String("filePath", partPath),
	)

	src, err := os.Open(partPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open assembled file: %w", err)
	}
	defer src.Close()

	pathResp, err := s.enqueueVideoUploadTask(userID, fileName, src, action)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue video upload task: %w", err)
	}

	return &pathResp, nil
}

func (s *fileService) DeleteVideoStream(userID string, fileName string) error {
	// remove any temp part file
	partPath := filepath.Join(os.TempDir(), "video_uploads", userID, fileName+".part")
	_ = os.Remove(partPath)

	key := fmt.Sprintf("%s/%s", userID, filepath.Base(fileName))
	if err := s.videoStorage.Delete(context.TODO(), key); err != nil {
		return fmt.Errorf("failed to delete video from streaming repo: %w", err)
	}
	return nil
}

// File upload and delete
func (s *fileService) UploadFile(userID string, filePath string, destination string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// ensure filename is clean
	fileName := filepath.Base(destination)
	key := fmt.Sprintf("%s/%s", userID, fileName)

	err = s.imageStorage.Put(context.TODO(), key, file, "application/octet-stream")
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Build object URL (use presigned for private buckets)
	url := s.imageStorage.BuildUrl(key)
	return url, nil
}

func (s *fileService) DeleteFile(userID string, fileName string) error {
	key := fmt.Sprintf("%s/%s", userID, filepath.Base(fileName))
	return s.imageStorage.Delete(context.TODO(), key)
}

func NewFileService(storage3rd *third_party_repository.ThirdPartyStorageRegistry, rabbitmq *rabbitmq.RabbitMQ) iservice.FileService {
	return &fileService{
		imageStorage: storage3rd.S3Storage,
		videoStorage: storage3rd.S3StreamStorage,
		rabbitmq:     rabbitmq,
	}
}

// Helpers
func (s *fileService) getTempDir(userID string) string {
	//return filepath.Join(os.TempDir(), userID) //prod
	return filepath.Join("tmp", userID) //test
}

func (s *fileService) getTempFilePath(userID string, fileName string) string {
	return filepath.Join(s.getTempDir(userID), fileName+".part")
}

func (s *fileService) appendChunkToFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
		}
	}(f)

	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}

func (s *fileService) enqueueVideoUploadTask(userID, fileName string, src *os.File, action *string) (responses.PathResponse, error) {
	// RabbitMQ handle this step
	tempPath := s.getTempFilePath(userID, fileName)
	key := fmt.Sprintf("%s/%s", userID, filepath.Base(fileName))

	url := s.videoStorage.BuildUrl(key)

	pathResp := responses.PathResponse{
		HostURL: url,
		//TempURL: host + "/" + tempPath, //TODO: optimize later
		TempURL: url,
	}

	videoMessage := &consumers.VideoUploadMessage{
		UserID:   userID,
		FilePath: tempPath,
		Key:      key,
		Action:   action,
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
	if err := videoProducer.Publish(context.Background(), payload); err != nil {
		zap.L().Error("Failed to publish video upload task", zap.Error(err))
		return pathResp, fmt.Errorf("failed to publish video upload task: %w", err)
	}

	zap.L().Info("✅ Video upload task enqueued",
		zap.String("userID", userID),
		zap.String("fileName", fileName),
		zap.String("tempPath", tempPath),
		zap.String("s3Key", key),
	)

	return pathResp, nil
}
