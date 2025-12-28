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

func (h *Handler) RegisterUserRoutes(r *gin.RouterGroup) {
	// User-level routes (no tenant context required)
	r.POST("/tenants", h.CreateTenant)
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

type CreateRequest struct {
	Name string `json:"name" binding:"required,max=255"`
}

func (h *Handler) CreateTenant(c *gin.Context) {
	// Get authenticated user ID from context (set by Auth middleware)
	userID, err := appContext.UserID(c.Request.Context())
	if err != nil || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse request body
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create tenant with user as owner
	tenant, err := h.service.CreateWithOwner(c.Request.Context(), req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}
