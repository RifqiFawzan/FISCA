package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

const (
	endpointCreateTransaction = "POST /transactions"
	endpointVoidTransaction   = "POST /transactions/void"
)

type transactionCreateResponse struct {
	Transaction service.TransactionDTO `json:"transaction"`
	Warnings    []service.Warning      `json:"warnings"`
}

func (h *Handler) createTransaction(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.CreateTransactionInput
	if !h.bind(c, &in) {
		return
	}
	key, ok := h.idempotencyKey(c, true)
	if !ok {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	replay, done, err := h.Svc.BeginIdempotency(ctx, h.Svc.Q, userID, key, endpointCreateTransaction, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	if done {
		replayResponse(c, replay)
		return
	}

	var out transactionCreateResponse
	err = h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		tx, warnings, err := h.Svc.CreateTransaction(ctx, q, userID, in, h.writeContext(c))
		if err != nil {
			return err
		}
		out = transactionCreateResponse{Transaction: tx, Warnings: warnings}
		envelope := httpx.Envelope{
			Data: out,
			Meta: &httpx.Meta{RequestID: httpx.RequestID(c)},
		}
		return h.Svc.CompleteIdempotency(ctx, q, userID, key, endpointCreateTransaction, http.StatusCreated, envelope)
	})
	if err != nil {
		h.Svc.AbortIdempotency(ctx, h.Svc.Q, userID, key, endpointCreateTransaction)
		h.fail(c, err)
		return
	}
	httpx.Created(c, out)
}

func (h *Handler) listTransactions(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	page, err := h.Svc.ListTransactions(c.Request.Context(), q, userID, service.ListTransactionsInput{
		Type:       c.Query("type"),
		CategoryID: c.Query("categoryId"),
		AccountID:  c.Query("accountId"),
		Q:          c.Query("q"),
		Period:     c.Query("period"),
		From:       c.Query("from"),
		To:         c.Query("to"),
		Cursor:     c.Query("cursor"),
		Limit:      queryInt(c, "limit", 20),
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OKWithMeta(c, page, &httpx.Meta{RequestID: httpx.RequestID(c), NextCursor: page.NextCursor})
}

func (h *Handler) searchTransactions(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	result, err := h.Svc.SearchTransactions(c.Request.Context(), q, userID, c.Query("q"), queryInt(c, "limit", 20))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OKWithMeta(c, result, &httpx.Meta{RequestID: httpx.RequestID(c), NextCursor: result.Next})
}

func (h *Handler) getTransaction(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	detail, err := h.Svc.GetTransaction(c.Request.Context(), q, userID, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, detail)
}

func (h *Handler) updateTransaction(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateTransactionInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.TransactionDTO
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		tx, err := h.Svc.UpdateTransaction(ctx, q, userID, c.Param("id"), in, h.writeContext(c))
		if err != nil {
			return err
		}
		out = tx
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}

type voidTransactionInput struct {
	Reason string `json:"reason"`
}

func (h *Handler) voidTransaction(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in voidTransactionInput
	if !h.bind(c, &in) {
		return
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
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}{ID: c.Param("id"), Reason: in.Reason}

	replay, done, err := h.Svc.BeginIdempotency(ctx, h.Svc.Q, userID, key, endpointVoidTransaction, payload)
	if err != nil {
		h.fail(c, err)
		return
	}
	if done {
		replayResponse(c, replay)
		return
	}

	var out service.TransactionDTO
	err = h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		tx, err := h.Svc.VoidTransaction(ctx, q, userID, payload.ID, payload.Reason, h.writeContext(c))
		if err != nil {
			return err
		}
		out = tx
		envelope := httpx.Envelope{Data: tx, Meta: &httpx.Meta{RequestID: httpx.RequestID(c)}}
		return h.Svc.CompleteIdempotency(ctx, q, userID, key, endpointVoidTransaction, http.StatusOK, envelope)
	})
	if err != nil {
		h.Svc.AbortIdempotency(ctx, h.Svc.Q, userID, key, endpointVoidTransaction)
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}
