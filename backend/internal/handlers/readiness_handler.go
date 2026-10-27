package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func DatabaseReadiness(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"database": gin.H{
					"status": "unavailable",
				},
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"database": gin.H{
					"status": "unavailable",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"database": gin.H{
				"status": "connected",
			},
		})
	}
}
