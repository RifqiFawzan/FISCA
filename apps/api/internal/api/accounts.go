package api

import (
	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

func (h *Handler) getMe(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	me, err := h.Svc.GetMe(c.Request.Context(), q, userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, me)
}

func (h *Handler) updateMe(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateMeInput
	if !h.bind(c, &in) {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	me, err := h.Svc.UpdateMe(c.Request.Context(), q, userID, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, me)
}

// ---------------------------------------------------------------------------
// accounts
// ---------------------------------------------------------------------------

func (h *Handler) listAccounts(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	accounts, err := h.Svc.ListAccounts(c.Request.Context(), q, userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, accounts)
}

func (h *Handler) createAccount(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.CreateAccountInput
	if !h.bind(c, &in) {
		return
	}
	if _, ok := h.ensureUser(c, userID); !ok {
		return
	}
	var created service.AccountDTO
	err := h.Svc.WithTx(c.Request.Context(), func(tx *generated.Queries) error {
		account, err := h.Svc.CreateAccount(c.Request.Context(), tx, userID, in)
		if err != nil {
			return err
		}
		created = account
		return nil
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.Created(c, created)
}

func (h *Handler) updateAccount(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateAccountInput
	if !h.bind(c, &in) {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	account, err := h.Svc.UpdateAccount(c.Request.Context(), q, userID, c.Param("id"), in)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, account)
}

func (h *Handler) setDefaultAccount(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	account, err := h.Svc.SetDefaultAccount(c.Request.Context(), q, userID, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, account)
}

func (h *Handler) archiveAccount(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	if err := h.Svc.ArchiveAccount(c.Request.Context(), q, userID, c.Param("id")); err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"archived": true})
}

// ---------------------------------------------------------------------------
// categories
// ---------------------------------------------------------------------------

func (h *Handler) listCategories(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	categories, err := h.Svc.ListCategories(c.Request.Context(), q, userID, c.Query("kind"))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, categories)
}

func (h *Handler) createCategory(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.CreateCategoryInput
	if !h.bind(c, &in) {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	category, err := h.Svc.CreateCategory(c.Request.Context(), q, userID, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.Created(c, category)
}

func (h *Handler) updateCategory(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	var in service.UpdateCategoryInput
	if !h.bind(c, &in) {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	category, err := h.Svc.UpdateCategory(c.Request.Context(), q, userID, c.Param("id"), in)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, category)
}

func (h *Handler) archiveCategory(c *gin.Context) {
	userID, ok := h.user(c)
	if !ok {
		return
	}
	q, ok := h.ensureUser(c, userID)
	if !ok {
		return
	}
	if err := h.Svc.ArchiveCategory(c.Request.Context(), q, userID, c.Param("id")); err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"archived": true})
}
