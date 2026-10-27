package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct{ DB *pgxpool.Pool }

type createUserRequest struct {
	FullName string `json:"full_name" binding:"required,max=150"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=12,max=128"`
	Role     string `json:"role" binding:"required,oneof=admin staff reviewer"`
}

type updateUserRequest struct {
	FullName *string `json:"full_name" binding:"omitempty,max=150"`
	Role     *string `json:"role" binding:"omitempty,oneof=admin staff reviewer"`
	IsActive *bool   `json:"is_active"`
}

func NewUserHandler(db *pgxpool.Pool) *UserHandler { return &UserHandler{DB: db} }

func (h *UserHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := h.DB.Query(ctx, `SELECT id, full_name, email, role, is_active, created_at FROM users ORDER BY created_at DESC, id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Daftar user gagal dimuat."})
		return
	}
	defer rows.Close()
	type userRecord struct {
		ID        int64     `json:"id"`
		FullName  string    `json:"full_name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
	}
	users := make([]userRecord, 0)
	for rows.Next() {
		var user userRecord
		if err := rows.Scan(&user.ID, &user.FullName, &user.Email, &user.Role, &user.IsActive, &user.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Daftar user gagal dibaca."})
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Daftar user gagal dibaca."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *UserHandler) Create(c *gin.Context) {
	var input createUserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, email, password (minimal 12 karakter), dan role valid wajib diisi."})
		return
	}
	name, email := strings.TrimSpace(input.FullName), strings.ToLower(strings.TrimSpace(input.Email))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama lengkap wajib diisi."})
		return
	}
	hash, err := auth.HashPassword(input.Password)
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
		IsActive bool   `json:"is_active"`
	}
	err = h.DB.QueryRow(ctx, `INSERT INTO users (full_name, email, password_hash, role) VALUES ($1,$2,$3,$4) RETURNING id, full_name, email, role, is_active`, name, email, hash, input.Role).Scan(&user.ID, &user.FullName, &user.Email, &user.Role, &user.IsActive)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar atau akun tidak dapat dibuat."})
		return
	}
	_, _ = h.DB.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, details) VALUES ($1,'user.created','user',$2,jsonb_build_object('role',$3))`, c.GetInt64("user_id"), user.ID, user.Role)
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *UserHandler) Update(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID user tidak valid."})
		return
	}
	var input updateUserRequest
	if err := c.ShouldBindJSON(&input); err != nil || (input.FullName == nil && input.Role == nil && input.IsActive == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kirim nama, role, atau status aktif yang valid."})
		return
	}
	if input.FullName != nil {
		trimmed := strings.TrimSpace(*input.FullName)
		if trimmed == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama lengkap tidak boleh kosong."})
			return
		}
		input.FullName = &trimmed
	}
	actorID := c.GetInt64("user_id")
	if userID == actorID && ((input.Role != nil && *input.Role != "admin") || (input.IsActive != nil && !*input.IsActive)) {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin tidak dapat menurunkan role atau menonaktifkan akunnya sendiri."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	tx, err := h.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Perubahan user gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialize admin changes so simultaneous updates cannot remove the last active admin.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(4981234)`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Perubahan user gagal diproses."})
		return
	}
	var oldRole string
	var oldActive bool
	err = tx.QueryRow(ctx, `SELECT role, is_active FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&oldRole, &oldActive)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User gagal dimuat."})
		return
	}
	newRole, newActive := oldRole, oldActive
	if input.Role != nil {
		newRole = *input.Role
	}
	if input.IsActive != nil {
		newActive = *input.IsActive
	}
	if oldRole == "admin" && oldActive && (newRole != "admin" || !newActive) {
		var activeAdmins int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE role='admin' AND is_active=TRUE`).Scan(&activeAdmins); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Status admin gagal diperiksa."})
			return
		}
		if activeAdmins <= 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "Minimal harus ada satu Admin aktif."})
			return
		}
	}
	var user struct {
		ID        int64     `json:"id"`
		FullName  string    `json:"full_name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
	}
	err = tx.QueryRow(ctx, `UPDATE users SET full_name=COALESCE($2,full_name), role=$3, is_active=$4, updated_at=NOW() WHERE id=$1 RETURNING id,full_name,email,role,is_active,created_at`, userID, input.FullName, newRole, newActive).Scan(&user.ID, &user.FullName, &user.Email, &user.Role, &user.IsActive, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Perubahan user gagal disimpan."})
		return
	}
	if oldActive && !newActive {
		if _, err = tx.Exec(ctx, `UPDATE user_sessions SET revoked_at=NOW() WHERE user_id=$1 AND revoked_at IS NULL`, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Sesi user gagal dinonaktifkan."})
			return
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, details) VALUES ($1,'user.updated','user',$2,jsonb_build_object('old_role',$3,'role',$4,'is_active',$5))`, actorID, userID, oldRole, newRole, newActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Audit perubahan user gagal disimpan."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Perubahan user gagal disimpan."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}
