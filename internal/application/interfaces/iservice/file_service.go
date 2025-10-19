package iservice

import "core-backend/internal/application/dto/responses"

type FileService interface {
	UploadFile(userID string, filePath string, destination string) (string, error)
	DeleteFile(userID string, fileName string) error

	//Streaming
	UploadVideoStream(userID string, fileName string, data []byte, isLastChunk bool, action *string) (*responses.PathResponse, error)
	DeleteVideoStream(userID string, fileName string) error
}
