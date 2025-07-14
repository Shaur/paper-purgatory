package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"paper/purgatory/service"
)

type controller struct {
	service service.PurgatoryService
}

type PurgatoryController interface {
	Get(ctx *gin.Context)
}

func Init(service service.PurgatoryService) PurgatoryController {
	return &controller{service: service}
}

func (c *controller) Get(ctx *gin.Context) {
	items := c.service.GetAll()
	ctx.JSON(http.StatusOK, items)
}
