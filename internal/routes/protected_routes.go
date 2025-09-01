package routes

import (
	"dice-sorensen-similarity-search/internal/auth"
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/constants"
	"dice-sorensen-similarity-search/internal/markdowndoc"
	"dice-sorensen-similarity-search/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func RegisterProtectedRoutes(r *gin.Engine, controllerRegistry map[int]any) {

	authGroup := r.Group("")

	authGroup.Use(middlewares.AuthHandler())
	{
		// bitbucket
		bitbucketApi := controllerRegistry[constants.Bitbucket].(bitbucket.Api)
		authGroup.GET("/bitbucket/markdowns", bitbucketApi.FetchMarkdownsFromBitbucket)

		// auth
		authApi := controllerRegistry[constants.Auth].(auth.Api)
		authGroup.GET("/markdown-doc/token", authApi.GetAuthToken)

		// markdown doc
		markdownDocApi := controllerRegistry[constants.MarkdownDoc].(markdowndoc.Api)
		authGroup.GET("/markdown-doc/navigation-items", markdownDocApi.GetNavigationItemsTrees)
		authGroup.GET("/markdown-doc/markdown/:name", markdownDocApi.GetMarkdownByName)
		authGroup.POST("/markdown-doc/markdown/search", markdownDocApi.GetMarkdownSearchTermMatches)
	}
}
