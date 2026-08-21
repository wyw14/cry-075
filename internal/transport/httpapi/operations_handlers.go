package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/middleware"
)

func (a API) uploadAttachment(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, err)
		return
	}
	opened, err := file.Open()
	if err != nil {
		fail(c, err)
		return
	}
	defer opened.Close()
	key, err := a.Attachments.Upload(c, file.Filename, file.Header.Get("Content-Type"), file.Size, opened, middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"storage_key": key})
}
func (a API) exportAudit(c *gin.Context) {
	var input domain.ExportRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	input.RequestedBy = middleware.GetActor(c).ID
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=audit.csv")
	if err := a.Operations.ExportAudit(c, input, middleware.GetActor(c), c.Writer); err != nil {
		fail(c, err)
		return
	}
}
func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
