package api

import (
	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

// GET /savings-goals
func (h *Handler) listSavingsGoals(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	goals, err := h.Svc.ListSavingsGoals(c.Request.Context(), q, userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, goals)
}

// POST /savings-goals
func (h *Handler) createSavingsGoal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.CreateSavingsGoalInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var created service.SavingsGoalDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		goal, err := h.Svc.CreateSavingsGoal(ctx, q, userID, in, h.writeContext(c))
		if err != nil {
			return err
		}
		created = goal
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.Created(c, created)
}

// GET /savings-goals/{id}?monthly=500000
func (h *Handler) getSavingsGoal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	goal, err := h.Svc.GetSavingsGoal(
		c.Request.Context(), q, userID, c.Param("id"), int64(queryInt(c, "monthly", 0)),
	)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, goal)
}

// PATCH /savings-goals/{id}
func (h *Handler) updateSavingsGoal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateSavingsGoalInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var updated service.SavingsGoalDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		goal, err := h.Svc.UpdateSavingsGoal(ctx, q, userID, c.Param("id"), in, h.writeContext(c))
		if err != nil {
			return err
		}
		updated = goal
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, updated)
}

// DELETE /savings-goals/{id}
func (h *Handler) archiveSavingsGoal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		return h.Svc.ArchiveSavingsGoal(ctx, q, userID, c.Param("id"), h.writeContext(c))
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"archived": true})
}

// GET /purchase-advisor?price=1500000
func (h *Handler) purchaseAdvisor(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	analysis, err := h.Svc.PurchaseAdvisor(
		c.Request.Context(), q, userID, int64(queryInt(c, "price", 0)),
	)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, analysis)
}
