package middlewares

import "github.com/gin-gonic/gin"

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		// cache-control directives (i.e., instructions):
		// no-store ... indicates that any caches of any kind (private / shared) should not store HTTP responses
		//
		// setting "no-store" is a cautious move.
		// if, for whatever reason, it makes sense to cache certain HTTP responses,
		// make sure you use the "private" directive for user-personalized content (e.g., login or cookies).
		// also, make sure to set a proper "max-age", and use a proper revalidation strategy
		//
		// see https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Cache-Control
		c.Header("Cache-Control", "no-store")

		c.Next()
	}
}
