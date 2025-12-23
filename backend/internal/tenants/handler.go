// internal/tenants/handler.go
package tenants

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Keep backward compatible route names for now
	r.GET("/organization", h.Get)
	r.PUT("/organization", h.Update)
}

func (h *Handler) Get(c *gin.Context) {
	// TODO: Will be updated to use tenant_id from context in Phase 2
	tenantID := c.GetString("org_id")

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

func (h *Handler) Update(c *gin.Context) {
	// TODO: Will be updated to use tenant_id from context in Phase 2
	tenantID := c.GetString("org_id")
	role := c.GetString("role")

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
