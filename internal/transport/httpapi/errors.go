package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/middleware"
)

type errorBody struct {
	Code        string                  `json:"code"`
	Message     string                  `json:"message"`
	FieldErrors []domain.FieldViolation `json:"field_errors"`
	RequestID   string                  `json:"request_id"`
}

func fail(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用"
	switch domain.ErrorCodeOf(err) {
	case domain.CodeNotFound:
		status, code, message = http.StatusNotFound, "NOT_FOUND", "资源不存在"
	case domain.CodeForbidden:
		status, code, message = http.StatusForbidden, "FORBIDDEN", "没有执行该操作的权限"
	case domain.CodeConflict, domain.CodeVersionConflict, domain.CodeDuplicateRequest:
		status, code, message = http.StatusConflict, "BUSINESS_CONFLICT", err.Error()
	case domain.CodeRuleViolation, domain.CodeInvalidReference, domain.CodeUnavailableBackup:
		status, code, message = http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION", err.Error()
	}
	fields := []domain.FieldViolation{}
	var validation *domain.ValidationError
	if errors.As(err, &validation) {
		status, code, message = http.StatusBadRequest, "VALIDATION_FAILED", "请求字段不合法"
		fields = validation.Violations
	}
	c.AbortWithStatusJSON(status, errorBody{Code: code, Message: message, FieldErrors: fields, RequestID: middleware.GetRequestID(c)})
}
