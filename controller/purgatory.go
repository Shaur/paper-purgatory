package controller

import (
	"fmt"
	"net/http"
	"os"
	"paper/purgatory/dto"
	"paper/purgatory/service"
	"paper/purgatory/utils"

	"github.com/gin-gonic/gin"
)

type controller struct {
	service service.PurgatoryService
}

type PurgatoryController interface {
	Get(ctx *gin.Context)

	UploadFile(ctx *gin.Context)

	AddMeta(ctx *gin.Context)
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

	dest, destPath, err := c.service.UploadTempFile(file)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Server error while upload file"})
		return
	}

	defer utils.HandleRemove(os.Remove, destPath)
	defer utils.HandleClose(dest.Close)

	err = c.service.Save(dest, file.Filename)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File processing error"})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *controller) AddMeta(ctx *gin.Context) {
	var meta dto.NewMeta
	if err := ctx.BindJSON(&meta); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	item := c.service.SaveMeta(meta)

	ctx.JSON(http.StatusOK, item)
}
