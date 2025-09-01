package routes

import (
	"dice-sorensen-similarity-search/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func InitRouter(engine *gin.Engine, controllerRegistry map[int]any) {
	InitMiddleware(engine)

	RegisterProtectedRoutes(engine, controllerRegistry)
	RegisterPublicRoutes(engine, controllerRegistry)
	RegisterUtilityRoutes(engine)
}

func InitMiddleware(engine *gin.Engine) {
	engine.Use(middlewares.CORSMiddleware())
}
