package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/middleware"
)

func (a API) preview(c *gin.Context) {
	var input domain.PreviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	result, err := a.Preview.Render(c, input)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a API) takedown(c *gin.Context) {
	var input struct {
		Reason domain.InvalidationReason `json:"reason" binding:"required,oneof=copyright_expired manual_invalidation emergency_takedown"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	result, err := a.Invalidation.EmergencyTakedown(c, domain.ID(c.Param("id")), input.Reason, middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a API) snapshot(c *gin.Context) {
	result, err := a.Releases.Snapshot(c, domain.ID(c.Param("id")), middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}
func (a API) publish(c *gin.Context) {
	var input struct {
		Reason string `json:"reason" binding:"required,min=2,max=500"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	result, err := a.Releases.Publish(c, domain.ID(c.Param("id")), input.Reason, middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}
func (a API) rollback(c *gin.Context) {
	result, err := a.Releases.Rollback(c, domain.ID(c.Param("id")), middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}
func (a API) schedule(c *gin.Context) {
	var input struct {
		Action         domain.ScheduleAction `json:"action" binding:"required,oneof=publish pause expire"`
		ExecuteAt      string                `json:"execute_at" binding:"required"`
		Version        int64                 `json:"version" binding:"required,min=1"`
		IdempotencyKey string                `json:"idempotency_key" binding:"required,max=128"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	result, err := a.Scheduling.Plan(c, domain.ID(c.Param("id")), input.Action, input.ExecuteAt, input.Version, input.IdempotencyKey, middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}
