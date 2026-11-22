package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

// RegisterPublic memasang rute yang harus jalan tanpa token: mustahil
// meminta token untuk mendapatkan token. Ditaruh di grup terpisah dari v1
// karena v1 memakai middleware.Auth pada seluruh anggotanya.
func (h *Handler) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/register", h.register)
	g.POST("/login", h.login)
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authPayload adalah respons register/login. Token dikembalikan sekali saja;
// setelah itu server hanya menyimpan hash-nya (0003_auth.sql).
type authPayload struct {
	Token     string          `json:"token"`
	ExpiresAt string          `json:"expiresAt"`
	User      service.UserDTO `json:"user"`
}

func authPayloadOf(result service.AuthResult) authPayload {
	return authPayload{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339),
		User:      result.User,
	}
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if !h.bind(c, &req) {
		return
	}
	result, err := h.Svc.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.Created(c, authPayloadOf(result))
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if !h.bind(c, &req) {
		return
	}
	result, err := h.Svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, authPayloadOf(result))
}

// logout mencabut sesi di balik token yang sedang dipakai. Berjalan di bawah
// middleware.Auth, jadi token sudah pasti valid pada titik ini.
func (h *Handler) logout(c *gin.Context) {
	if err := h.Svc.Logout(c.Request.Context(), httpx.BearerToken(c)); err != nil {
		h.fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}
