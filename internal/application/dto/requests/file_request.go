package requests

import (
	"mime/multipart"
	"strings"

	"github.com/go-playground/validator/v10"
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

// endregion
