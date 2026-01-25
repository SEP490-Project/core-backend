package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-playground/validator/v10"

	"core-backend/internal/application/interfaces/iservice"

	"github.com/gin-gonic/gin"
)

type S3Handler struct {
	fileService iservice.FileService
	validator   *validator.Validate
}

func NewS3Handler(fileService iservice.FileService) *S3Handler {
	v := validator.New()
	_ = v.RegisterValidation("resolutions", requests.ValidateResolutions)
	v.RegisterStructValidation(requests.ValidateFileFilterRequest, requests.FileFilterRequest{})
	return &S3Handler{
		fileService: fileService,
		validator:   v,
	}
}

// UploadFile godoc
//
//	@Summary	Upload files to S3
//	@Tags		Files
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		files	formData	file	true	"Files to upload"
//	@Param		userId	formData	string	true	"User ID"
//	@Success	200		{object}	map[string][]string
//	@Failure	400		{object}	map[string]string
//	@Router		/api/v1/files/upload [post]
func (h *S3Handler) UploadFile(c *gin.Context) {
	userID := c.PostForm("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form data"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "files are required"})
		return
	}

	userTmpDir := fmt.Sprintf("./tmp/%s", userID)
	if err := os.MkdirAll(userTmpDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user tmp directory"})
		return
	}

	var uploadedURLs []string
	for _, fileHeader := range files {
		// Generate unique timestamped file name
		timestamp := time.Now().Format("20060102_150405")
		newFileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
		finalPath := fmt.Sprintf("%s/%s", userTmpDir, newFileName)

		// Save uploaded file
		if err := c.SaveUploadedFile(fileHeader, finalPath); err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + fileHeader.Filename})
			return
		}

		defer func(path string) { _ = os.Remove(path) }(finalPath)

		url, err := h.fileService.UploadFile(c.Request.Context(), userID, finalPath, newFileName)
		if err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "FileRepositoryfailed to upload file: " + fileHeader.Filename + ", " + err.Error()})
			return
		}

		// Cleanup tmp file after upload
		_ = os.Remove(finalPath)

		uploadedURLs = append(uploadedURLs, url)
	}

	c.JSON(http.StatusOK, gin.H{"urls": uploadedURLs})
}

// DeleteFile godoc
//
//	@Summary	Delete a file from S3
//	@Tags		Files
//	@Param		userId		query		string	true	"User ID"
//	@Param		filename	path		string	true	"File name"
//	@Success	200			{object}	map[string]string
//	@Failure	400			{object}	map[string]string
//	@Router		/api/v1/files/{filename} [delete]
func (h *S3Handler) DeleteFile(c *gin.Context) {
	userID := c.Query("userId")
	filename := c.Param("filename")
	if userID == "" || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId and filename are required"})
		return
	}
	err := h.fileService.DeleteFile(c.Request.Context(), userID, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}

// region: ============== UploadVideoChunk ==============

// UploadVideoChunk godoc
//
//	@Summary	Upload a video chunk (streaming upload)
//	@Tags		Files
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		userId			formData	string	true	"User ID"
//	@Param		fileName		formData	string	true	"Final file name (eg myvideo.mp4)"
//	@Param		isLastChunk		formData	boolean	true	"Whether this is the final chunk"
//	@Param		chunk			formData	file	true	"Chunk file"
//	@Param		isHls			formData	boolean	false	"Convert to HLS"
//	@Param		resolutions		formData	string	false	"Comma-separated list of resolutions for HLS (options: 144p,240p,360p,480p,720p,1080p,1440p)"
//	@Param		segmentDuration	formData	int		false	"HLS segment duration in seconds (default 10)"
//	@Success	200				{object}	map[string]string
//	@Failure	400				{object}	map[string]string
//	@Failure	500				{object}	map[string]string
//	@Router		/api/v1/files/videos/upload-chunk [post]
func (h *S3Handler) UploadVideoChunk(c *gin.Context) {
	var req requests.UploadVideoChunkRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	fh, err := req.Chunk.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open chunk"})
		return
	}
	defer fh.Close()

	data, err := io.ReadAll(fh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk data"})
		return
	}

	paths, err := h.fileService.UploadVideoStream(c.Request.Context(), &req, &data)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.IsLastChunk {
		resp := responses.SuccessResponse("Video receives", ptr.Int(http.StatusOK), paths)
		c.JSON(http.StatusOK, resp)
	} else {
		resp := responses.SuccessResponse("Chunk received", ptr.Int(http.StatusPartialContent), nil)
		c.JSON(http.StatusPartialContent, resp)
	}
}

// DeleteVideo godoc
//
//	@Summary	Delete uploaded video
//	@Tags		Files
//	@Param		userId		query		string	true	"User ID"
//	@Param		fileName	query		string	true	"File name"
//	@Success	200			{object}	map[string]string
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/api/v1/files/videos [delete]
func (h *S3Handler) DeleteVideo(c *gin.Context) {
	userID := c.Query("userId")
	fileName := c.Query("fileName")
	if userID == "" || fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId and fileName are required"})
		return
	}
	if err := h.fileService.DeleteVideoStream(c.Request.Context(), userID, fileName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// endregion

// region: ============== GET Methods ==============

/* PaginationRequest
UploadedBy *uuid.UUID       `form:"uploaded_by" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
StorageKey *string          `form:"storage_key" example:"files/example.jpg"`
Keyword    *string          `form:"keyword" validate:"omitempty,min=1" example:"example"`
MimeType   *string          `form:"mime_type" example:"image/jpeg"`
MinSize    *int64           `form:"min_size" validate:"omitempty,min=0" example:"1048576"`
MaxSize    *int64           `form:"max_size" validate:"omitempty,min=0,gtefield=MinSize" example:"1048576"`
FromDate   *string          `form:"from_date" validate:"omitempty,datetime=2006-01-02" example:"2023-01-01"`
ToDate     *string          `form:"to_date" validate:"omitempty,datetime=2006-01-02" example:"2023-12-31"`
Status     *enum.FileStatus `form:"status" validate:"omitempty,oneof='PENDING' 'UPLOADING' 'UPLOADED' 'FAILED'" example:"UPLOADED"` */

// GetFileByFilter godoc
//
//	@Summary	Get file info by filter
//	@Tags		Files
//	@Param		uploaded_by	query	string	false	"UUID of the user who uploaded the file"
//	@Param		storage_key	query	string	true	"Storage key of the file"
//	@Param		keyword		query	string	false	"Keyword to search in file names"
//	@Param		mime_type	query	string	false	"MIME type of the file"
//	@Param		min_size	query	int64	false	"Minimum file size in bytes"
//	@Param		max_size	query	int64	false	"Maximum file size in bytes"
//	@Param		from_date	query	string	false	"Start date for upload date range (YYYY-MM-DD)"
//	@Param		to_date		query	string	false	"End date for upload date range (YYYY-MM-DD)"
//	@Param		status		query	string	false	"Status of the file (PENDING, UPLOADING, UPLOADED, FAILED)"
//	@Param		page		query	int		false	"Page number for pagination"
//	@Param		limit		query	int		false	"Number of items per page for pagination"
//	@Produce	json
//	@Success	200	{object}	responses.FilePaginationResponse
//	@Failure	400	{object}	responses.APIResponse
//	@Failure	500	{object}	responses.APIResponse
//	@Router		/api/v1/files [get]
func (h *S3Handler) GetFileByFilter(c *gin.Context) {
	var filterReq requests.FileFilterRequest
	if err := c.ShouldBindQuery(&filterReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.validator.Struct(&filterReq); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	files, total, err := h.fileService.GetFileByFilter(c.Request.Context(), &filterReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := responses.NewPaginationResponse(
		"Files retrieved successfully",
		http.StatusOK,
		files,
		responses.Pagination{
			Total: total,
			Page:  filterReq.Page,
			Limit: filterReq.Limit,
		},
	)
	c.JSON(http.StatusOK, resp)
}

// GetFileDetailByS3Key godoc
//
//	@Summary	Get file detail by S3 key
//	@Tags		Files
//	@Param		key	path	string	true	"S3 storage key of the file"
//	@Produce	json
//	@Success	200	{object}	responses.FileDetailResponse
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/files/{key} [get]
func (h *S3Handler) GetFileDetailByS3Key(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	file, err := h.fileService.GetFileByS3Key(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, file)
}

// endregion
