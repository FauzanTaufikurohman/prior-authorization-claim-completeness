package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionDuration = 12 * time.Hour

type AuthHandler struct {
	DB *pgxpool.Pool
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	FullName string `json:"full_name" binding:"required,max=150"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=12,max=128"`
}

func NewAuthHandler(db *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{DB: db}
}

// Register creates public accounts as staff. Role changes are restricted to admins.
func (h *AuthHandler) Register(c *gin.Context) {
	var input registerRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, email, atau password tidak valid. Password minimal 12 karakter."})
		return
	}

	name := strings.TrimSpace(input.FullName)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama lengkap wajib diisi."})
		return
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password minimal 12 karakter."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	var user struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	err = h.DB.QueryRow(ctx, `
		INSERT INTO users (full_name, email, password_hash, role)
		VALUES ($1, $2, $3, 'staff')
		RETURNING id, full_name, email, role
	`, name, email, passwordHash).Scan(&user.ID, &user.FullName, &user.Email, &user.Role)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar atau akun tidak dapat dibuat."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user, "message": "Pendaftaran berhasil. Silakan masuk dengan akun Staff Anda."})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input loginRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email atau password tidak valid.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user struct {
		ID           int64
		FullName     string
		Email        string
		PasswordHash string
		Role         string
		IsActive     bool
	}

	err := h.DB.QueryRow(ctx, `
		SELECT id, full_name, email, password_hash, role, is_active
		FROM users
		WHERE lower(email) = lower($1)
	`, strings.TrimSpace(input.Email)).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
	)

	if err != nil || !user.IsActive ||
		!auth.VerifyPassword(user.PasswordHash, input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Email atau password salah.",
		})
		return
	}

	token, tokenHash, err := auth.GenerateSessionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat sesi.",
		})
		return
	}

	expiresAt := time.Now().UTC().Add(sessionDuration)

	_, err = h.DB.Exec(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, tokenHash, expiresAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat sesi.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"token_type": "Bearer",
		"expires_at": expiresAt,
		"user": gin.H{
			"id":        user.ID,
			"full_name": user.FullName,
			"email":     user.Email,
			"role":      user.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetString("session_token")
	tokenHash := auth.HashSessionToken(token)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, err := h.DB.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengakhiri sesi.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout berhasil.",
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        c.GetInt64("user_id"),
			"full_name": c.GetString("full_name"),
			"email":     c.GetString("email"),
			"role":      c.GetString("role"),
		},
	})
}
