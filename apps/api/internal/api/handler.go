// Package api contains thin HTTP handlers. Every financial rule lives in
// internal/service; handlers only bind, authorize, and render envelopes.
package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/ai"
	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

// Handler holds the shared dependencies for all routes.
type Handler struct {
	Svc *service.Service
}

// New builds a Handler.
func New(svc *service.Service) *Handler { return &Handler{Svc: svc} }

// Register mounts every v1 route on an already authenticated group.
func (h *Handler) Register(v1 *gin.RouterGroup) {
	v1.GET("/me", h.getMe)
	v1.PATCH("/me", h.updateMe)

	v1.GET("/accounts", h.listAccounts)
	v1.POST("/accounts", h.createAccount)
	v1.PATCH("/accounts/:id", h.updateAccount)
	v1.POST("/accounts/:id/default", h.setDefaultAccount)
	v1.DELETE("/accounts/:id", h.archiveAccount)

	v1.GET("/categories", h.listCategories)
	v1.POST("/categories", h.createCategory)
	v1.PATCH("/categories/:id", h.updateCategory)
	v1.DELETE("/categories/:id", h.archiveCategory)

	v1.POST("/transactions", h.createTransaction)
	v1.GET("/transactions", h.listTransactions)
	v1.GET("/transactions/search", h.searchTransactions)
	v1.GET("/transactions/:id", h.getTransaction)
	v1.PATCH("/transactions/:id", h.updateTransaction)
	v1.POST("/transactions/:id/void", h.voidTransaction)

	v1.GET("/analytics/dashboard", h.dashboard)
	v1.GET("/analytics/financial-summary", h.financialSummary)
	v1.GET("/analytics/largest-transactions", h.largestTransactions)
	v1.GET("/analytics/unusual-spending", h.unusualSpending)
	v1.GET("/analytics/spending-comparison", h.spendingComparison)
	v1.GET("/analytics/cash-flow", h.cashFlow)

	v1.GET("/budgets", h.listBudgets)
	v1.POST("/budgets", h.createBudget)
	v1.GET("/budgets/:id", h.getBudget)
	v1.PATCH("/budgets/:id", h.updateBudget)
	v1.GET("/budgets/:id/status", h.budgetStatus)

	v1.POST("/budget-proposals", h.createProposal)
	v1.GET("/budget-proposals", h.listProposals)
	v1.GET("/budget-proposals/:id", h.getProposal)
	v1.PATCH("/budget-proposals/:id", h.updateProposal)
	v1.POST("/budget-proposals/:id/approve", h.approveProposal)
	v1.POST("/budget-proposals/:id/reject", h.rejectProposal)

	v1.GET("/savings-goals", h.listSavingsGoals)
	v1.POST("/savings-goals", h.createSavingsGoal)
	v1.GET("/savings-goals/:id", h.getSavingsGoal)
	v1.PATCH("/savings-goals/:id", h.updateSavingsGoal)
	v1.DELETE("/savings-goals/:id", h.archiveSavingsGoal)
	v1.GET("/purchase-advisor", h.purchaseAdvisor)

	v1.POST("/assistant/messages", h.assistantMessage)
	v1.GET("/assistant/conversations", h.listConversations)
	v1.GET("/assistant/conversations/:id", h.getConversation)
	v1.DELETE("/assistant/conversations/:id", h.deleteConversation)

	v1.POST("/speech/transcriptions", h.transcribe)

	// Berada di dalam v1 (terautentikasi): logout butuh token yang sah untuk
	// dicabut. Register/login sengaja TIDAK di sini — lihat RegisterPublic.
	v1.POST("/auth/logout", h.logout)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (h *Handler) user(c *gin.Context) (uuid.UUID, bool) {
	raw := httpx.UserID(c)
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		httpx.Unauthorized(c, "invalid user identity")
		return uuid.Nil, false
	}
	return id, true
}

// ensureUser provisions the profile (and defaults) on first login, then returns
// a query handle for the request.
func (h *Handler) ensureUser(c *gin.Context, userID uuid.UUID) (*generated.Queries, bool) {
	q := h.Svc.Q
	if _, err := h.Svc.EnsureUser(c.Request.Context(), q, userID, "", ""); err != nil {
		h.fail(c, err)
		return nil, false
	}
	return q, true
}

func (h *Handler) writeContext(c *gin.Context) service.WriteContext {
	return service.WriteContext{
		Actor:     "user",
		RequestID: httpx.RequestID(c),
	}
}

// fail renders a domain error with the right status code.
func (h *Handler) fail(c *gin.Context, err error) {
	if errors.Is(err, ai.ErrUnavailable) {
		httpx.Fail(c, http.StatusServiceUnavailable, httpx.CodeAIUnavailable,
			"Fitur AI belum dikonfigurasi", nil)
		return
	}
	domain := service.AsError(err)
	httpx.Fail(c, domain.Status, domain.Code, domain.Message, domain.Details)
}

func (h *Handler) bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, httpx.CodeValidation,
			"Body JSON tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func (h *Handler) withTx(c *gin.Context, fn func(q *generated.Queries) error) bool {
	if err := h.Svc.WithTx(c.Request.Context(), fn); err != nil {
		h.fail(c, err)
		return false
	}
	return true
}

func (h *Handler) idempotencyKey(c *gin.Context, required bool) (string, bool) {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		if required {
			httpx.Fail(c, http.StatusUnprocessableEntity, httpx.CodeValidation,
				"Header Idempotency-Key wajib untuk operasi ini", nil)
			return "", false
		}
		return "", true
	}
	if len(key) > 128 {
		httpx.Fail(c, http.StatusUnprocessableEntity, httpx.CodeValidation,
			"Idempotency-Key maksimal 128 karakter", nil)
		return "", false
	}
	return key, true
}

// replay writes a stored idempotent response.
func replayResponse(c *gin.Context, res service.IdempotencyReplay) {
	if len(res.Body) == 0 {
		c.Status(res.StatusCode)
		return
	}
	c.Data(res.StatusCode, "application/json; charset=utf-8", res.Body)
}
