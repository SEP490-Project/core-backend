package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"mime/multipart"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type UploadVideoChunkRequest struct {
	UserID          string                `form:"userId" validate:"required,uuid"`
	FileName        string                `form:"fileName" validate:"required"`
	IsLastChunk     bool                  `form:"isLastChunk"`
	Chunk           *multipart.FileHeader `form:"chunk" validate:"required"`
	IsHLS           bool                  `form:"isHls"`
	Resolutions     string                `form:"resolutions" validate:"omitempty,resolutions"`
	SegmentDuration int                   `form:"segmentDuration" validate:"omitempty,min=1,max=60"`
}

func (r *UploadVideoChunkRequest) GetResolutions() []string {
	if r.Resolutions == "" {
		return nil
	}
	return strings.Split(r.Resolutions, ",")
}

type FileFilterRequest struct {
	PaginationRequest
	UploadedBy *uuid.UUID       `form:"uploaded_by" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	StorageKey *string          `form:"storage_key" example:"files/example.jpg"`
	Keyword    *string          `form:"keyword" validate:"omitempty,min=1" example:"example"`
	MimeType   *string          `form:"mime_type" example:"image/jpeg"`
	MinSize    *int64           `form:"min_size" validate:"omitempty,min=0" example:"1048576"`
	MaxSize    *int64           `form:"max_size" validate:"omitempty,min=0,gtefield=MinSize" example:"1048576"`
	FromDate   *string          `form:"from_date" validate:"omitempty,datetime=2006-01-02" example:"2023-01-01"`
	ToDate     *string          `form:"to_date" validate:"omitempty,datetime=2006-01-02" example:"2023-12-31"`
	Status     *enum.FileStatus `form:"status" validate:"omitempty,oneof='PENDING' 'UPLOADING' 'UPLOADED' 'FAILED'" example:"UPLOADED"`
}

// region: ======== Custom validators =======

// ValidateResolutions checks if resolutions are in format "720p,1080p"
// Currently supports 144p, 240p, 360p, 480p, 720p, 1080p, 1440p
func ValidateResolutions(fl validator.FieldLevel) bool {
	supportedResolutions := map[string]bool{
		"144p":  true,
		"240p":  true,
		"360p":  true,
		"480p":  true,
		"720p":  true,
		"1080p": true,
		"1440p": true,
	}

	resolutions := fl.Field().String()
	if resolutions == "" {
		return true
	}
	parts := strings.SplitSeq(resolutions, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if !strings.HasSuffix(part, "p") {
			return false
		}
		if _, ok := supportedResolutions[part]; !ok {
			return false
		}
	}
	return true
}

func ValidateFileFilterRequest(sl validator.StructLevel) {
	filterRequest := sl.Current().Interface().(FileFilterRequest)

	if filterRequest.FromDate != nil {
		if _, err := utils.ParseLocalTime(*filterRequest.FromDate, utils.DateFormat); err != nil {
			sl.ReportError(filterRequest.FromDate, "from_date", "FromDate", "datetime", "Invalid date format, expected YYYY-MM-DD")
		}
	}
	if filterRequest.ToDate != nil {
		if _, err := utils.ParseLocalTime(*filterRequest.ToDate, utils.DateFormat); err != nil {
			sl.ReportError(filterRequest.ToDate, "to_date", "ToDate", "datetime", "Invalid date format, expected YYYY-MM-DD")
		}
	}
}

// endregion
