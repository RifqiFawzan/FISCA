package api

import (
	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
)

func (h *Handler) dashboard(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.Dashboard(c.Request.Context(), q, userID, c.Query("period"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) financialSummary(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.FinancialSummary(c.Request.Context(), q, userID, c.Query("period"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) largestTransactions(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.LargestTransactions(
		c.Request.Context(), q, userID,
		c.Query("period"), c.Query("type"), c.Query("categoryId"),
		queryInt(c, "limit", 10),
	)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) unusualSpending(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.UnusualSpending(c.Request.Context(), q, userID, c.Query("period"), c.Query("categoryId"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) spendingComparison(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.SpendingComparison(c.Request.Context(), q, userID, c.Query("period"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) cashFlow(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.CashFlow(c.Request.Context(), q, userID, c.Query("period"), c.Query("bucket"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}
