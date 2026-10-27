package handlers

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
)

var npiNumberPattern = regexp.MustCompile(`^\d{10}$`)

type NPIHandler struct{}

func NewNPIHandler() *NPIHandler {
	return &NPIHandler{}
}

func (h *NPIHandler) Lookup(c *gin.Context) {
	number := c.Param("number")
	if !npiNumberPattern.MatchString(number) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "NPI harus terdiri dari 10 digit angka.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 9*time.Second)
	defer cancel()

	provider, err := services.LookupNPI(ctx, number)
	if err != nil {
		if errors.Is(err, services.ErrNPIRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "NPI tidak ditemukan di CMS NPI Registry."})
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "CMS NPI Registry tidak merespons tepat waktu. Coba lagi atau lanjutkan dengan input manual."})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Lookup CMS NPI Registry gagal. Coba lagi atau lanjutkan dengan input manual."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": provider})
}
