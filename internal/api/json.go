package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

const (
	Success string = "success" //The request ended successfully
	Error   string = "error"   //The request ended with error - check the message field
	//Running      string = "running"      //The request was accepted but the job which for which is called is in running state
	//Pending      string = "pending"      //The request was accepted but is in a pending state
	//Unauthorized string = "unauthorized" //The request ended because you are not allowed to access that resource
)

type GenericRequest struct {
	Data map[string]interface{} `json:"data"`
}

func NewGenericResponse(status string, message string, data interface{}) gin.H {
	return gin.H{
		"status":  status,
		"message": message,
		"data":    data,
	}
}

func NewErrorResponse(message string) gin.H {
	return gin.H{
		"status":  Error,
		"message": message,
		"data":    gin.H{},
	}
}

func NewErrorResponsef(format string, a ...interface{}) gin.H {
	return gin.H{
		"status":  Error,
		"message": fmt.Sprintf(format, a...),
		"data":    gin.H{},
	}
}

func (genericRequest *GenericRequest) DecodeDataTo(output interface{}) error {
	err := mapstructure.Decode(genericRequest.Data, &output)
	if err != nil {
		return err
	}
	return nil
}

func (genericRequest *GenericRequest) Load(input []byte) error {
	err := json.Unmarshal(input, &genericRequest)
	if err != nil {
		return err
	}
	return nil
}

type RestJsonRequest struct {
	Data interface{} `json:"data"`
}

type RestJsonResponse struct {
	Status  string      `json:"status" example:"success"`
	Message string      `json:"message" example:"The request was sent successfully"`
	Data    interface{} `json:"data"`
}

type RestJsonErrorResponse struct {
	Status  string      `json:"status" example:"error"`
	Message string      `json:"message" example:"No host or ciid defined"`
	Data    interface{} `json:"data"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RestJsonLoginResponse struct {
	Status  string `json:"status" example:"success"`
	Message string `json:"message" example:" "`
	Data    string `json:"data" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
}
