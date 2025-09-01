package routes

import (
	"dice-sorensen-similarity-search/internal/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterUtilityRoutes(r *gin.Engine) {
	r.GET("/heartbeat", controllers.GetHeartBeat)
	r.GET("/status", controllers.GetStatus)
}
