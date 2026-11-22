// Package service holds the business rules described in plan.md §8. Handlers
// stay thin: they bind input, open transactions, call this package, and render.
package service

import (
	"errors"
	"net/http"
)

// Error codes shared with the HTTP layer (plan.md §7.1).
const (
	CodeValidation         = "VALIDATION_ERROR"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeIdempotency        = "IDEMPOTENCY_MISMATCH"
	CodeBudgetStale        = "BUDGET_STALE"
	CodeRateLimited        = "RATE_LIMITED"
	CodeAIUnavailable      = "AI_UNAVAILABLE"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeInternal           = "INTERNAL"
)

// Error is a domain error carrying the HTTP status and stable error code.
type Error struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *Error) Error() string { return e.Message }

// Validation builds a 422 error.
func Validation(message string, details any) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: CodeValidation, Message: message, Details: details}
}

// Invalid builds a 422 error with a single field message.
func Invalid(field, message string) *Error {
	return Validation(message, map[string]string{"field": field})
}

// NotFound builds a 404 error.
func NotFound(entity string) *Error {
	return &Error{Status: http.StatusNotFound, Code: CodeNotFound, Message: entity + " tidak ditemukan"}
}

// Conflict builds a 409 error.
func Conflict(message string, details any) *Error {
	return &Error{Status: http.StatusConflict, Code: CodeConflict, Message: message, Details: details}
}

// BudgetStale builds a 409 BUDGET_STALE error.
func BudgetStale(message string, details any) *Error {
	return &Error{Status: http.StatusConflict, Code: CodeBudgetStale, Message: message, Details: details}
}

// IdempotencyMismatch builds a 409 IDEMPOTENCY_MISMATCH error.
func IdempotencyMismatch() *Error {
	return &Error{
		Status:  http.StatusConflict,
		Code:    CodeIdempotency,
		Message: "Idempotency-Key sudah dipakai untuk request yang berbeda",
	}
}

// AIUnavailable builds a 503 error.
func AIUnavailable(message string) *Error {
	return &Error{Status: http.StatusServiceUnavailable, Code: CodeAIUnavailable, Message: message}
}

// AsError converts an error to a domain error, wrapping unknown errors as 500.
func AsError(err error) *Error {
	var domain *Error
	if errors.As(err, &domain) {
		return domain
	}
	return &Error{Status: http.StatusInternalServerError, Code: CodeInternal, Message: "internal server error"}
}
