package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"

	"github.com/google/uuid"
)

// PathResponse represents the response containing file path information.
type PathResponse struct {
	HostURL string
	TempURL string
}

type FileDetailResponse struct {
	ID          uuid.UUID         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string            `json:"file_name" example:"example.jpg"`
	AltText     *string           `json:"alt_text,omitempty" example:"Alt text"`
	URL         string            `json:"url" example:"https://example.com/files/example.jpg"`
	StorageKey  string            `json:"storage_key" example:"files/example.jpg"`
	MimeType    string            `json:"mime_type" example:"image/jpeg"`
	Size        int64             `json:"size" example:"204800"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Status      enum.FileStatus   `json:"status" example:"COMPLETED"`
	ErrorReason *string           `json:"error_reason,omitempty" example:"Error details if any"`
	UploadedAt  string            `json:"uploaded_at,omitempty" example:"2023-10-01 12:00:00"`
	UploadedBy  uuid.UUID         `json:"uploaded_by,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	UpdatedAt   string            `json:"updated_at" example:"2023-10-01 12:00:00"`
}

func (FileDetailResponse) ToResponse(fileModel *model.File) *FileDetailResponse {
	var metadata map[string]string
	if fileModel.Metadata != nil {
		rawMetadata, err := json.Marshal(fileModel.Metadata)
		if err == nil {
			_ = json.Unmarshal(rawMetadata, &metadata)
		}
	}

	return &FileDetailResponse{
		ID:          fileModel.ID,
		Name:        fileModel.Name,
		AltText:     fileModel.AltText,
		URL:         fileModel.URL,
		StorageKey:  fileModel.StorageKey,
		MimeType:    fileModel.MimeType,
		Size:        fileModel.Size,
		Metadata:    metadata,
		Status:      fileModel.Status,
		ErrorReason: fileModel.ErrorReason,
		UploadedAt:  utils.FormatLocalTime(fileModel.UploadedAt, utils.TimeFormat),
		UploadedBy:  utils.DerefPtr(fileModel.UploadedBy, uuid.Nil),
		UpdatedAt:   utils.FormatLocalTime(&fileModel.UpdatedAt, utils.TimeFormat),
	}
}

type FileListResponse struct {
	ID          uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string          `json:"file_name" example:"example.jpg"`
	URL         string          `json:"url" example:"https://example.com/files/example.jpg"`
	MimeType    string          `json:"mime_type" example:"image/jpeg"`
	Size        int64           `json:"size" example:"204800"`
	Status      enum.FileStatus `json:"status" example:"COMPLETED"`
	ErrorReason *string         `json:"error_reason,omitempty" example:"Error details if any"`
	UploadedAt  string          `json:"uploaded_at,omitempty" example:"2023-10-01 12:00:00"`
	UploadedBy  uuid.UUID       `json:"uploaded_by,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	UpdatedAt   string          `json:"updated_at" example:"2023-10-01 12:00:00"`
}

func (FileListResponse) ToResponse(fileModel *model.File) *FileListResponse {
	return &FileListResponse{
		ID:          fileModel.ID,
		Name:        fileModel.Name,
		URL:         fileModel.URL,
		MimeType:    fileModel.MimeType,
		Size:        fileModel.Size,
		Status:      fileModel.Status,
		ErrorReason: fileModel.ErrorReason,
		UploadedAt:  utils.FormatLocalTime(fileModel.UploadedAt, utils.TimeFormat),
		UploadedBy:  utils.DerefPtr(fileModel.UploadedBy, uuid.Nil),
		UpdatedAt:   utils.FormatLocalTime(&fileModel.UpdatedAt, utils.TimeFormat),
	}
}

func (FileListResponse) ToResponseList(fileModels []model.File) []FileListResponse {
	if len(fileModels) == 0 {
		return []FileListResponse{}
	}
	responseList := make([]FileListResponse, len(fileModels))
	for i, fileModel := range fileModels {
		responseList[i] = utils.DerefPtr(FileListResponse{}.ToResponse(&fileModel), FileListResponse{})
	}
	return responseList
}

type FilePaginationResponse PaginationResponse[FileListResponse]
