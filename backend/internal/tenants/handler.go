// internal/tenants/handler.go
package tenants

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Keep backward compatible route names for now
	r.GET("/tenant", h.GetTenant)
	r.PUT("/tenant", h.UpdateTenant)
}

func (h *Handler) GetTenant(c *gin.Context) {
	tenantID := appContext.MustTenantID(c.Request.Context())

	tenant, err := h.service.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

type UpdateRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) UpdateTenant(c *gin.Context) {
	tenantID := appContext.MustTenantID(c.Request.Context())
	role := appContext.UserRole(c.Request.Context())

	// Only owners/admins can update
	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant, err := h.service.Update(c.Request.Context(), tenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}
