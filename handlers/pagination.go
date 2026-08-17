package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/pagination"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

func parseIncludeTotal(c *gin.Context) (bool, bool) {
	raw := strings.TrimSpace(c.Query("include_total"))
	if raw == "" {
		return false, true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "include_total debe ser true o false")
		return false, false
	}
	return value, true
}

type nameCursor struct {
	Name string `json:"n"`
	ID   int    `json:"i"`
}

type clientCursor struct {
	LastName string `json:"a"`
	Name     string `json:"n"`
	ID       int    `json:"i"`
}

type timeCursor struct {
	CreatedAt time.Time `json:"t"`
	ID        int       `json:"i"`
}

type planPriorityCursor struct {
	StateRank int       `json:"r"`
	CreatedAt time.Time `json:"t"`
	ID        int       `json:"i"`
}

type reservationCursor struct {
	Local string    `json:"l"`
	Date  time.Time `json:"d"`
	Time  string    `json:"t"`
	ID    int       `json:"i"`
}

func parsePagination(c *gin.Context) (pagination.Request, bool) {
	request, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return pagination.Request{}, false
	}
	return request, true
}

func decodePaginationCursor(c *gin.Context, request pagination.Request, scope string, filters any, target any) bool {
	if request.Cursor == "" {
		return true
	}
	if err := pagination.Decode(request.Cursor, scope, filters, target); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}
