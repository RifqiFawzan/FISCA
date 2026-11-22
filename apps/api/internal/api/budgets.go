package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

const endpointApproveProposal = "POST /budget-proposals/approve"

func (h *Handler) listBudgets(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.ListBudgets(c.Request.Context(), q, userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) createBudget(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.CreateBudgetInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.BudgetDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		budget, err := h.Svc.CreateBudget(ctx, q, userID, in, h.writeContext(c))
		if err != nil {
			return err
		}
		out = budget
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.Created(c, out)
}

func (h *Handler) getBudget(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.GetBudget(c.Request.Context(), q, userID, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) updateBudget(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateBudgetInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.BudgetDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		budget, err := h.Svc.UpdateBudget(ctx, q, userID, c.Param("id"), in, h.writeContext(c))
		if err != nil {
			return err
		}
		out = budget
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}

func (h *Handler) budgetStatus(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.GetBudgetStatus(c.Request.Context(), q, userID, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) createProposal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.CreateProposalInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.BudgetProposalDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		proposal, err := h.Svc.CreateProposal(ctx, q, userID, in, h.writeContext(c))
		if err != nil {
			return err
		}
		out = proposal
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.Created(c, out)
}

func (h *Handler) listProposals(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.ListProposals(c.Request.Context(), q, userID, c.Query("status"), queryInt(c, "limit", 20))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) getProposal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.GetProposal(c.Request.Context(), q, userID, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) updateProposal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateProposalInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.BudgetProposalDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		proposal, err := h.Svc.UpdateProposal(ctx, q, userID, c.Param("id"), in)
		if err != nil {
			return err
		}
		out = proposal
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}

type approveProposalInput struct {
	BaseVersion int32 `json:"baseVersion"`
}

func (h *Handler) approveProposal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in approveProposalInput
	// baseVersion is optional in the body because the proposal carries one; an
	// empty body is accepted, a malformed one is not.
	if c.Request.ContentLength > 0 {
		if !h.bind(c, &in) {
			return
		}
	}
	key, ok := h.idempotencyKey(c, true)
	if !ok {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	payload := struct {
		ID          string `json:"id"`
		BaseVersion int32  `json:"baseVersion"`
	}{ID: c.Param("id"), BaseVersion: in.BaseVersion}

	replay, done, err := h.Svc.BeginIdempotency(ctx, h.Svc.Q, userID, key, endpointApproveProposal, payload)
	if err != nil {
		h.fail(c, err)
		return
	}
	if done {
		replayResponse(c, replay)
		return
	}

	var out service.BudgetDTO
	err = h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		budget, err := h.Svc.ApproveProposal(ctx, q, userID, payload.ID, payload.BaseVersion, h.writeContext(c))
		if err != nil {
			return err
		}
		out = budget
		envelope := httpx.Envelope{Data: budget, Meta: &httpx.Meta{RequestID: httpx.RequestID(c)}}
		return h.Svc.CompleteIdempotency(ctx, q, userID, key, endpointApproveProposal, http.StatusOK, envelope)
	})
	if err != nil {
		h.Svc.AbortIdempotency(ctx, h.Svc.Q, userID, key, endpointApproveProposal)
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}

func (h *Handler) rejectProposal(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.BudgetProposalDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		proposal, err := h.Svc.RejectProposal(ctx, q, userID, c.Param("id"), h.writeContext(c))
		if err != nil {
			return err
		}
		out = proposal
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}
