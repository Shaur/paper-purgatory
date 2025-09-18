package controller

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"paper/purgatory/service"
	"paper/purgatory/utils"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type controller struct {
	service service.PurgatoryService
}

type PurgatoryController interface {
	Get(ctx *gin.Context)

	UploadFile(ctx *gin.Context)
}

func Init(service service.PurgatoryService) PurgatoryController {
	return &controller{service: service}
}

func (c *controller) Get(ctx *gin.Context) {
	items := c.service.GetAll()
	ctx.JSON(http.StatusOK, items)
}

func (c *controller) UploadFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File is not presented"})
		return
	}

	dest, destPath, err := uploadTempFile(file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Server error while upload file"})
		return
	}

	defer utils.HandleRemove(os.Remove, destPath)

	err = c.service.Save(dest, file.Filename)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File processing error"})
		return
	}

	ctx.Status(http.StatusOK)
}

func uploadTempFile(source *multipart.FileHeader) (*os.File, string, error) {
	src, err := source.Open()
	if err != nil {
		return nil, "", err
	}

	filename := source.Filename
	destPath := filepath.Join("files", filename)
	dest, err := os.Create(destPath)
	if err != nil {
		return nil, "", err
	}

	_, err = io.Copy(dest, src)
	if err != nil {
		return nil, "", err
	}

	return dest, destPath, nil
}
