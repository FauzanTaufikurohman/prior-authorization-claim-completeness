package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RequireAuth(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Authentication service unavailable.",
			})
			return
		}

		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") ||
			len(parts[1]) < 40 || len(parts[1]) > 100 {
			unauthorized(c)
			return
		}

		token := parts[1]
		tokenHash := auth.HashSessionToken(token)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		var user struct {
			ID       int64
			FullName string
			Email    string
			Role     string
		}

		err := db.QueryRow(ctx, `
			SELECT u.id, u.full_name, u.email, u.role
			FROM user_sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.token_hash = $1
			  AND s.revoked_at IS NULL
			  AND s.expires_at > NOW()
			  AND u.is_active = TRUE
		`, tokenHash).Scan(
			&user.ID,
			&user.FullName,
			&user.Email,
			&user.Role,
		)

		if err != nil {
			unauthorized(c)
			return
		}

		c.Set("user_id", user.ID)
		c.Set("full_name", user.FullName)
		c.Set("email", user.Email)
		c.Set("role", user.Role)
		c.Set("session_token", token)

		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")

		for _, allowed := range allowedRoles {
			if role == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Anda tidak memiliki izin untuk mengakses resource ini.",
		})
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "Sesi tidak valid atau sudah berakhir. Silakan login kembali.",
	})
}
