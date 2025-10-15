package handler

import (
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"os"

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
