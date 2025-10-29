package handler

import (
	"core-backend/internal/application/dto/responses"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/smithy-go/ptr"

	"core-backend/internal/application/interfaces/iservice"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

type S3Handler struct {
	fileService iservice.FileService
}

func NewS3Handler(fileService iservice.FileService) *S3Handler {
	return &S3Handler{fileService: fileService}
}

// UploadFile godoc
//
//	@Summary	Upload files to S3
//	@Tags		files
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

	var urls []string
	for _, fileHeader := range files {
		tempPath := "/tmp/" + uuid.New().String() + "_" + fileHeader.Filename
		if err = c.SaveUploadedFile(fileHeader, tempPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + fileHeader.Filename})
			return
		}
		defer func(path string) { _ = os.Remove(path) }(tempPath)

		url, err := h.fileService.UploadFile(userID, tempPath, fileHeader.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file: " + fileHeader.Filename + ", " + err.Error()})
			return
		}
		urls = append(urls, url)
	}

	c.JSON(http.StatusOK, gin.H{"urls": urls})
}

// DeleteFile godoc
//
//	@Summary	Delete a file from S3
//	@Tags		files
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
	err := h.fileService.DeleteFile(userID, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}

// Videos

// UploadVideoChunk godoc
//
//	@Summary	Upload a video chunk (streaming upload)
//	@Tags		files
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		userId		formData	string	true	"User ID"
//	@Param		fileName	formData	string	true	"Final file name (eg myvideo.mp4)"
//	@Param		isLastChunk	formData	boolean	true	"Whether this is the final chunk"
//	@Param		chunk		formData	file	true	"Chunk file"
//	@Success	200			{object}	map[string]string
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/api/v1/files/videos/upload-chunk [post]
func (h *S3Handler) UploadVideoChunk(c *gin.Context) {
	userID := c.PostForm("userId")
	fileName := c.PostForm("fileName")
	isLastStr := c.PostForm("isLastChunk")

	if userID == "" || fileName == "" || isLastStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId, fileName and isLastChunk are required"})
		return
	}

	isLast, err := strconv.ParseBool(isLastStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "isLastChunk must be boolean"})
		return
	}

	fileHeader, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk file is required"})
		return
	}

	fh, err := fileHeader.Open()
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

	paths, err := h.fileService.UploadVideoStream(c.Request.Context(), userID, fileName, &data, isLast, nil)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if isLast {
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
//	@Tags		files
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
