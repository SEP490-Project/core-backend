package iservice

type FileService interface {
	UploadFile(userID string, filePath string, destination string) (string, error)
	DeleteFile(userID string, fileName string) error
}
