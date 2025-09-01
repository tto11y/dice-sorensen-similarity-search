package routes

import (
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/constants"
	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(r *gin.Engine, controllerRegistry map[int]any) {
	//r.GET("/something", controllers.Something)
	r.POST("/hook", func(c *gin.Context) {
		bitbucketApi := controllerRegistry[constants.Bitbucket].(bitbucket.Api)
		bitbucketApi.FetchMarkdownsFromBitbucket(c)
	})
}
