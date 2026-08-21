package domain

import "errors"

type ErrorCode string

const (
	CodeInternal          ErrorCode = "INTERNAL_ERROR"
	CodeNotFound          ErrorCode = "NOT_FOUND"
	CodeForbidden         ErrorCode = "FORBIDDEN"
	CodeConflict          ErrorCode = "BUSINESS_CONFLICT"
	CodeRuleViolation     ErrorCode = "BUSINESS_RULE_VIOLATION"
	CodeValidation        ErrorCode = "VALIDATION_FAILED"
	CodeVersionConflict   ErrorCode = "VERSION_CONFLICT"
	CodeDuplicateRequest  ErrorCode = "DUPLICATE_REQUEST"
	CodeInvalidReference  ErrorCode = "INVALID_REFERENCE"
	CodeUnavailableBackup ErrorCode = "NO_ELIGIBLE_FALLBACK"
)

type BusinessError struct {
	code    ErrorCode
	message string
}

func newBusinessError(code ErrorCode, message string) *BusinessError {
	return &BusinessError{code: code, message: message}
}

func (e *BusinessError) Error() string   { return e.message }
func (e *BusinessError) Code() ErrorCode { return e.code }

var (
	ErrNotFound          = newBusinessError(CodeNotFound, "resource not found")
	ErrConflict          = newBusinessError(CodeConflict, "business conflict")
	ErrInvalidTransition = newBusinessError(CodeRuleViolation, "invalid state transition")
	ErrVersionConflict   = newBusinessError(CodeVersionConflict, "optimistic version conflict")
	ErrForbidden         = newBusinessError(CodeForbidden, "permission denied")
	ErrDuplicateRequest  = newBusinessError(CodeDuplicateRequest, "idempotency key reused with different input")
	ErrInvalidReference  = newBusinessError(CodeInvalidReference, "invalid business reference")
	ErrNoFallback        = newBusinessError(CodeUnavailableBackup, "no eligible fallback package")
)

type FieldViolation struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string   { return "validation failed" }
func (e *ValidationError) Code() ErrorCode { return CodeValidation }

func ErrorCodeOf(err error) ErrorCode {
	var coded interface{ Code() ErrorCode }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return CodeInternal
}
