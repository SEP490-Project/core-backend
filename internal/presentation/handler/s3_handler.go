package handler

import (
	"core-backend/internal/application/interfaces/iservice"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

type S3Handler struct {
	fileService iservice.FileService
}

func NewS3Handler(fileService iservice.FileService) *S3Handler {
	return &S3Handler{fileService: fileService}
}

// UploadFile godoc
// @Summary Upload a file to S3
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param userId formData string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/files/upload [post]
func (h *S3Handler) UploadFile(c *gin.Context) {
	userID := c.PostForm("userId")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	tempPath := "/tmp/" + file.Filename
	if err = c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	defer func() { _ = os.Remove(tempPath) }()
	url, err := h.fileService.UploadFile(userID, tempPath, file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// DeleteFile godoc
// @Summary Delete a file from S3
// @Tags files
// @Param userId query string true "User ID"
// @Param filename path string true "File name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/files/{filename} [delete]
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
