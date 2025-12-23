package users

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jalil32/toggle/internal/tenants"
)

type Handler struct {
	service       *Service
	tenantService *tenants.Service
}

func NewHandler(service *Service, tenantService *tenants.Service) *Handler {
	return &Handler{
		service:       service,
		tenantService: tenantService,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/tenants", h.ListMyTenants)
	r.PUT("/active-tenant", h.SetActiveTenant)
}

// ListMyTenants returns all tenants that the authenticated user belongs to
func (h *Handler) ListMyTenants(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	tenants, err := h.tenantService.ListUserTenants(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenants"})
		return
	}

	c.JSON(http.StatusOK, tenants)
}

type SetActiveTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

// SetActiveTenant updates the user's last active tenant
func (h *Handler) SetActiveTenant(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req SetActiveTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user has access to this tenant
	role, err := h.tenantService.GetMembership(c.Request.Context(), userID, req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify tenant access"})
		return
	}

	if role == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this tenant"})
		return
	}

	// Update last active tenant
	err = h.service.UpdateLastActiveTenant(c.Request.Context(), userID, req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update active tenant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "active tenant updated",
		"tenant_id": req.TenantID,
	})
}
