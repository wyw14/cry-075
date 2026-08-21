package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/domain"
)

const ActorKey = "actor"

func Actor() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Actor-ID"))
		role := domain.Role(strings.TrimSpace(c.GetHeader("X-Actor-Role")))
		if id == "" || !validRole(role) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "AUTH_REQUIRED", "message": "请提供有效的本地操作者身份", "field_errors": []any{}, "request_id": GetRequestID(c)})
			return
		}
		c.Set(ActorKey, domain.Actor{ID: domain.ID(id), DisplayName: id, Role: role})
		c.Next()
	}
}
func GetActor(c *gin.Context) domain.Actor {
	value, _ := c.Get(ActorKey)
	actor, _ := value.(domain.Actor)
	return actor
}
func validRole(role domain.Role) bool {
	switch role {
	case domain.RoleOperator, domain.RoleReviewer, domain.RolePublisher, domain.RoleRightsAdmin, domain.RoleAuditor, domain.RoleScheduler:
		return true
	default:
		return false
	}
}
