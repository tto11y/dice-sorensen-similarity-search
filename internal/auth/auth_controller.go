package auth

import (
	"context"
	"dice-sorensen-similarity-search/internal/api"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/middlewares"
	"dice-sorensen-similarity-search/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Api defines the set of authentication-related endpoints exposed by the system.
//
// @Summary Authentication API
type Api interface {

	// Login to self-service
	Login(c *gin.Context)

	// RefreshToken creates a new access token after validating the old one
	RefreshToken(c *gin.Context)

	// CreatePasswordHash creates a hashed password which can then be used in the configuration
	CreatePasswordHash(c *gin.Context)

	// GetAuthToken validates credentials and issues a token without session context.
	GetAuthToken(c *gin.Context)
}

// Controller wires environment dependencies with authentication service methods.
// It fulfills the Api interface and delegates business logic to AuthService.
type Controller struct {
	*environment.Env
	*AuthService
}

// ensure Controller implements Api
var _ Api = &Controller{}

func (ac *Controller) Login(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ac.LogErrorf(nil, "Error reading login info: %v", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, api.NewErrorResponse("Error reading login info"))
		return
	}

	request := api.GenericRequest{}
	err = request.Load(body)
	if err != nil {
		ac.LogErrorf(nil, "Error loading request data: %v", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, api.NewErrorResponse("Error reading login info"))
		return
	}

	user := models.User{}
	err = request.DecodeDataTo(&user)
	if err != nil {
		ac.LogErrorf(nil, "Error loading user data: %v", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, api.NewErrorResponse("Error reading user info"))
		return
	}
	user.Prepare()
	err = user.Validate()
	if err != nil {
		ac.LogErrorf(nil, "Error validating user: %v", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, api.NewErrorResponsef("Error validating User: %v", err))
		return
	}

	err = ac.DoLogin(&user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.NewErrorResponse("Login not successful"))
		return
	}

	//issue token
	token, _, err := middlewares.GenerateToken(context.Background(), []byte(middlewares.SigningKey), 0, user.Username, []string{"admin"})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, api.NewErrorResponse("Error creating JWT"))
		return
	}
	c.JSON(http.StatusOK, api.NewGenericResponse(api.Success, "", token))
}

func (ac *Controller) RefreshToken(c *gin.Context) {
	tokenHeader := c.Request.Header.Get("Authorization")

	b := "Bearer "
	t := strings.Split(tokenHeader, b)
	if len(t) < 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "An authorization token was not supplied"})
		c.Abort()
		return
	}

	token, err := middlewares.ValidateToken(t[1], middlewares.SigningKey)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, err.Error())
		return
	}

	claims := token.Claims.(*middlewares.CimClaims)
	claims.ExpiresAt = time.Now().Add(43200 * time.Second).Unix()
	claims.IssuedAt = time.Now().Unix()

	// Create the token
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := newToken.SignedString([]byte(middlewares.SigningKey))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, api.NewErrorResponse("Error refreshing JWT"))
		return
	}
	c.JSON(http.StatusOK, api.NewGenericResponse(api.Success, "", tokenString))

}

func (ac *Controller) CreatePasswordHash(c *gin.Context) {
	password := c.Param("pw")
	hashPw, hashErr := models.Hash(password)
	if hashErr != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("an error occurred"))
		return
	} else {
		c.JSON(http.StatusOK, api.NewGenericResponse(api.Success, "your encrypted (bcrypt) password", string(hashPw)))
	}
}

// GetAuthToken validates credentials and issues a token without session context.
//
// @ID getAuthToken
// @Summary Issue authentication token
// @Tags auth
// @Router /markdown-doc/token [get]
// @Param context header string true "Containing the HTTP header "Authorization"; its value is used to obtain the token"
// @Success		200	{object}	api.RestJsonResponse{data=string}
// @Failure 401 {object} api.RestJsonResponse{data=string}
func (ac *Controller) GetAuthToken(c *gin.Context) {
	ac.LogInfo(nil, "getting auth token")

	genericToken := c.GetHeader("Authorization")
	if genericToken == "" {
		ac.LogErrorf(nil, "missing Authorization header")
		c.AbortWithStatusJSON(http.StatusForbidden, api.NewErrorResponsef("no token provided"))
		return
	}

	if genericToken != "generic" {
		ac.LogErrorf(nil, "the initial token does not match \\'generic\\'; invalid token provided")
		c.AbortWithStatusJSON(http.StatusForbidden, api.NewErrorResponsef("invalid token"))
		return
	}

	token, expiresAt, err := middlewares.GenerateToken(context.Background(), []byte(middlewares.SigningKey), 0, genericToken, []string{"admin"})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, api.NewErrorResponse("Error creating JWT"))
		return
	}

	ac.LogInfof(nil, "successfully generated token: %s", token)
	ac.LogInfof(nil, "successfully generated token; it expires in: %ds", expiresAt.Second())
	c.JSON(http.StatusOK, api.NewGenericResponse(api.Success, "", token))
}
