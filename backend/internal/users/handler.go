package users

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
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

// TenantResponse represents a tenant in API responses
type TenantResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ListMyTenants returns all tenants that the authenticated user belongs to
func (h *Handler) ListMyTenants(c *gin.Context) {
	userID := appContext.MustUserID(c.Request.Context())

	memberships, err := h.tenantService.ListUserTenants(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenants"})
		return
	}

	// Convert memberships to tenant responses
	tenants := make([]TenantResponse, len(memberships))
	for i, m := range memberships {
		tenants[i] = TenantResponse{
			ID:   m.TenantID,
			Name: m.TenantName,
			Slug: m.TenantSlug,
			Role: m.Role,
		}
	}

	c.JSON(http.StatusOK, tenants)
}

type SetActiveTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

// SetActiveTenant updates the user's last active tenant
func (h *Handler) SetActiveTenant(c *gin.Context) {
	userID := appContext.MustUserID(c.Request.Context())

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
