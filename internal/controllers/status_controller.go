package controllers

import (
	"dice-sorensen-similarity-search/internal/api"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetHeartBeat(c *gin.Context) {
	c.AbortWithStatus(http.StatusOK)
}

func GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, api.NewGenericResponse(api.Success, "running", nil))
}
