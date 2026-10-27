package routes

import (
	"net/http"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/handlers"
	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(pool *pgxpool.Pool) *gin.Engine {
	router := gin.Default()

	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
		api.GET("/health/ready", handlers.DatabaseReadiness(pool))

		authHandler := handlers.NewAuthHandler(pool)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/register", authHandler.Register)

		protected := api.Group("")
		protected.Use(middleware.RequireAuth(pool))
		{
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/auth/me", authHandler.Me)
			protected.GET("/providers/npi/:number", handlers.NewNPIHandler().Lookup)
			users := handlers.NewUserHandler(pool)
			adminUsers := protected.Group("/users")
			adminUsers.Use(middleware.RequireRoles("admin"))
			{
				adminUsers.GET("", users.List)
				adminUsers.POST("", users.Create)
				adminUsers.PATCH("/:id", users.Update)
			}

			pa := handlers.NewPARequestHandler(pool)
			workflow := handlers.NewPAWorkflowHandler(pool)
			checklistVersions := handlers.NewChecklistVersionHandler(pool)
			protected.GET("/checklist-options", checklistVersions.Options)
			documents := handlers.NewDocumentHandler(pool)
			checklists := handlers.NewChecklistHandler(pool)
			fields := handlers.NewExtractedFieldHandler(pool)
			requests := protected.Group("/pa-requests")
			{
				requests.GET("", pa.List)
				requests.POST("", middleware.RequireRoles("staff"), pa.Create)
				requests.GET("/:id", pa.GetByID)
				requests.PUT("/:id", pa.Update)
				requests.DELETE("/:id", pa.Delete)

				requests.GET("/:id/documents", documents.List)
				requests.POST("/:id/documents", documents.Upload)
				requests.POST("/:id/documents/:documentId/parse", documents.Parse)

				requests.GET("/:id/fields", fields.List)
				requests.PATCH("/:id/fields/:fieldId", fields.Update)

				requests.POST("/:id/checklist/evaluate", checklists.Evaluate)
				requests.GET("/:id/checklist/results", checklists.Results)
				requests.POST("/:id/transitions", workflow.Transition)
				requests.GET("/:id/status-history", workflow.History)
			}
			adminChecklists := protected.Group("/admin/checklist-versions")
			adminChecklists.Use(middleware.RequireRoles("admin"))
			{
				adminChecklists.GET("", checklistVersions.List)
				adminChecklists.POST("", checklistVersions.Create)
				adminChecklists.POST("/:id/activate", checklistVersions.Activate)
			}
		}

		api.GET(
			"/admin/access-check",
			middleware.RequireAuth(pool),
			middleware.RequireRoles("admin"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "Admin access granted.",
				})
			},
		)
	}

	return router
}
