package iservice

import (
	"context"
	"core-backend/internal/application/dto/responses"
)

type FileService interface {
	UploadFile(userID string, filePath string, destination string) (string, error)
	DeleteFile(userID string, fileName string) error

	//Streaming
	UploadVideoStream(ctx context.Context, userID string, fileName string, data *[]byte, isLastChunk bool, action *string) (*responses.PathResponse, error)
	DeleteVideoStream(ctx context.Context, userID string, fileName string) error
}
