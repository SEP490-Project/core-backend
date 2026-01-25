package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

type FileService interface {
	UploadFile(ctx context.Context, userID string, filePath string, destination string) (string, error)
	DeleteFile(ctx context.Context, userID string, fileName string) error

	//Streaming
	UploadVideoStream(ctx context.Context, req *requests.UploadVideoChunkRequest, data *[]byte) (*responses.PathResponse, error)
	DeleteVideoStream(ctx context.Context, userID string, fileName string) error

	// GET methods
	GetFileByFilter(ctx context.Context, filterRequest *requests.FileFilterRequest) ([]responses.FileListResponse, int64, error)
	GetFileByS3Key(ctx context.Context, key string) (*responses.FileDetailResponse, error)
}
