package handler

import (
	"core-backend/internal/application/dto/responses"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/smithy-go/ptr"

	"core-backend/internal/application/interfaces/iservice"

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

// endregion
