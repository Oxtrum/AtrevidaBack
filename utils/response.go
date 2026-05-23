package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Code    int         `json:"code" example:"200"`
	Data    interface{} `json:"data"`
	Error   bool        `json:"error" example:"false"`
	Message *string     `json:"message" example:"operacion realizada correctamente"`
	Status  string      `json:"status" example:"OK"`
}

func Respond(c *gin.Context, httpStatus int, data interface{}, message ...string) {
	var msg *string
	if len(message) > 0 && message[0] != "" {
		m := message[0]
		msg = &m
	}

	c.JSON(httpStatus, APIResponse{
		Code:    httpStatus,
		Data:    data,
		Error:   false,
		Message: msg,
		Status:  httpStatusText(httpStatus),
	})
}

func RespondError(c *gin.Context, httpStatus int, message string) {
	msg := message
	c.JSON(httpStatus, APIResponse{
		Code:    httpStatus,
		Data:    nil,
		Error:   true,
		Message: &msg,
		Status:  httpStatusText(httpStatus),
	})
}

// RespondMultiStatus se usa cuando algunos slots OK y otros fallaron (207).
func RespondMultiStatus(c *gin.Context, data interface{}, message string) {
	msg := message
	c.JSON(http.StatusMultiStatus, APIResponse{
		Code:    http.StatusMultiStatus,
		Data:    data,
		Error:   true, // parcialmente fallido
		Message: &msg,
		Status:  "PARTIAL",
	})
}

func httpStatusText(code int) string {
	switch code {
	case http.StatusOK:
		return "OK"
	case http.StatusCreated:
		return "CREATED"
	case http.StatusMultiStatus:
		return "MULTI_STATUS"
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	default:
		return http.StatusText(code)
	}
}

// Timestamp devuelve la hora actual.
func Timestamp() string {
	return time.Now().Format(time.RFC3339)
}
