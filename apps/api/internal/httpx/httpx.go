// Package httpx implements the response envelope described in plan.md §7.1.
//
//	success: {"data": ..., "meta": {...}}
//	failure: {"error": {"code": "...", "message": "...", "details": {...}, "requestId": "..."}}
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error codes shared between backend and frontend (plan.md §7.1).
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

// RequestIDKey is the gin context key holding the request id.
const RequestIDKey = "request_id"

// UserIDKey is the gin context key holding the authenticated user id.
const UserIDKey = "user_id"

// TokenKey is the gin context key holding the raw bearer token. Handler yang
// harus bertindak pada token itu sendiri (logout mencabut sesi) memakainya,
// sehingga header tidak perlu diparse ulang di lapisan handler.
const TokenKey = "bearer_token"

// Meta is the standard success metadata block.
type Meta struct {
	RequestID  string `json:"requestId,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
	Total      *int64 `json:"total,omitempty"`
}

// Envelope is the standard success response.
type Envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// ErrorBody is the standard error payload.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// ErrorEnvelope is the standard failure response.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// OK writes a success envelope with the request id attached.
func OK(c *gin.Context, data any) {
	OKWithMeta(c, data, &Meta{RequestID: RequestID(c)})
}

// OKWithMeta writes a success envelope with explicit metadata.
func OKWithMeta(c *gin.Context, data any, meta *Meta) {
	if meta != nil && meta.RequestID == "" {
		meta.RequestID = RequestID(c)
	}
	c.JSON(http.StatusOK, Envelope{Data: data, Meta: meta})
}

// Created writes a 201 success envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Data: data, Meta: &Meta{RequestID: RequestID(c)}})
}

// Fail writes an error envelope. It terminates the gin handler chain.
func Fail(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorEnvelope{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: RequestID(c),
		},
	})
}

// Unauthorized is a shorthand for a 401 error envelope.
func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, message, nil)
}

// RequestID returns the request id stored by the request-id middleware.
func RequestID(c *gin.Context) string {
	return c.GetString(RequestIDKey)
}

// UserID returns the authenticated user id, or an empty string when absent.
func UserID(c *gin.Context) string {
	return c.GetString(UserIDKey)
}

// BearerToken returns the raw bearer token of this request, or an empty
// string when the route did not run the auth middleware.
func BearerToken(c *gin.Context) string {
	return c.GetString(TokenKey)
}
