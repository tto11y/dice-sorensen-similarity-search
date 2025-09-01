package middlewares

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"strings"
	"time"
)

var (
	SigningKey = "79tesfUO0vy!U1wl7c8&EavOzmO2#W"
)

func AuthHandler(authRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.Request.Header.Get("Authorization")

		// let initial request pass
		path := c.Request.URL.Path
		if path == "/markdown-doc/token" && token == "generic" {
			c.Next()
			return
		}

		// Check if toke in correct format
		// ie Bearer xx03xllasx
		b := "Bearer "
		if !strings.Contains(token, b) {
			if len(token) <= 0 {
				c.JSON(403, gin.H{"message": "Your request is not authorized."})
			} else {
				c.JSON(403, gin.H{"message": "Your request is not authorized. Are you missing the prefix 'Bearer'?"})
			}
			c.Abort()
			return
		}
		t := strings.Split(token, b)
		if len(t) < 2 {
			c.JSON(403, gin.H{"message": "An authorization token was not supplied"})
			c.Abort()
			return
		}

		// Validate token
		_, err := ValidateToken(t[1], SigningKey)
		if err != nil {
			c.JSON(403, gin.H{"message": "Invalid authorization token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

type CimClaims struct {
	UserId   uint     `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.StandardClaims
}

func GenerateToken(ctx context.Context, key []byte, userId uint, username string, roles []string) (string, time.Time, error) {

	expiresAt := time.Now().Add(12 * time.Hour)
	claims := CimClaims{
		userId,
		username,
		roles,
		jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "dice-sorensen-search",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(key)
	return tokenString, expiresAt, err
}

func ValidateToken(tokenString string, key string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CimClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(key), nil
	})

	return token, err
}
