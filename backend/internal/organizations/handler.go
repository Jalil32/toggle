// internal/organizations/handler.go
package organizations

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
	r.GET("/organization", h.Get)
	r.PUT("/organization", h.Update)
}

func (h *Handler) Get(c *gin.Context) {
	orgID := c.GetString("org_id")

	org, err := h.service.GetByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, org)
}

type UpdateRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) Update(c *gin.Context) {
	orgID := c.GetString("org_id")
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

	org, err := h.service.Update(c.Request.Context(), orgID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, org)
}
