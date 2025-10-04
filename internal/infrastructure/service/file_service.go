// Package service
package service

import (
	"context"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/application/interfaces/iservice"
	"fmt"
	"os"
	"path/filepath"
)

type fileService struct {
	repo irepository_third_party.S3Repository
}

func (s *fileService) UploadFile(userId string, filePath string, destination string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// ensure filename is clean
	fileName := filepath.Base(destination)
	key := fmt.Sprintf("%s/%s", userID, fileName)

	err = s.repo.Put(context.TODO(), key, file, "application/octet-stream")
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Build object URL (use presigned for private buckets)
	url := s.repo.BuildUrl(key)
	return url, nil
}

func (s *fileService) DeleteFile(userId string, fileName string) error {
	key := fmt.Sprintf("%s/%s", userId, filepath.Base(fileName))
	return s.repo.Delete(context.TODO(), key)
}

func NewFileService(repo irepository_third_party.S3Repository) iservice.FileService {
	return &fileService{
		repo: repo,
	}
}
