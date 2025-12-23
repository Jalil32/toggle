package projects

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
	r.POST("/projects", h.Create)
	r.GET("/projects", h.List)
	r.GET("/projects/:id", h.GetByID)
	r.DELETE("/projects/:id", h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Will be updated to use tenant_id from context in Phase 2
	tenantID := c.GetString("org_id")

	project, err := h.service.Create(c.Request.Context(), tenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (h *Handler) List(c *gin.Context) {
	// TODO: Will be updated to use tenant_id from context in Phase 2
	tenantID := c.GetString("org_id")

	projects, err := h.service.ListByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, projects)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")

	project, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Check tenant ownership
	// TODO: Will be updated to use tenant_id from context in Phase 2
	if project.TenantID != c.GetString("org_id") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Verify ownership first
	project, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	// TODO: Will be updated to use tenant_id from context in Phase 2
	if project.TenantID != c.GetString("org_id") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
