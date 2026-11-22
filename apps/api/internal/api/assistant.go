package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

func (h *Handler) assistantMessage(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.AssistantRequest
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	ctx := c.Request.Context()
	var out service.AssistantResponse
	err := h.Svc.WithTx(ctx, func(q *generated.Queries) error {
		response, err := h.Svc.HandleMessage(ctx, q, userID, in, h.writeContext(c))
		if err != nil {
			return err
		}
		out = response
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, out)
}

func (h *Handler) listConversations(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	data, err := h.Svc.ListConversations(c.Request.Context(), q, userID, queryInt(c, "limit", 20))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, data)
}

func (h *Handler) getConversation(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	messages, err := h.Svc.GetConversation(c.Request.Context(), q, userID, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, messages)
}

func (h *Handler) deleteConversation(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	if err := h.Svc.DeleteConversation(c.Request.Context(), q, userID, c.Param("id")); err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *Handler) transcribe(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}

	// 25 MB is the provider's hard limit; reject before buffering.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 25<<20)
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, httpx.CodeValidation,
			"File audio wajib dikirim pada field 'audio'", gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	transcript, err := h.Svc.TranscribeAudio(c.Request.Context(), header.Filename, file)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"transcript": transcript,
		"durationMs": 0,
	})
}
