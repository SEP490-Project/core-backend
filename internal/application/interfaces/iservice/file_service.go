package iservice

type FileService interface {
	UploadFile(userId string, filePath string, destination string) (string, error)
	DeleteFile(userId string, fileName string) error
}
