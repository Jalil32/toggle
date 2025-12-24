package projects

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
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

	tenantID := appContext.MustTenantID(c.Request.Context())

	project, err := h.service.Create(c.Request.Context(), tenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (h *Handler) List(c *gin.Context) {
	tenantID := appContext.MustTenantID(c.Request.Context())

	projects, err := h.service.ListByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, projects)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	tenantID := appContext.MustTenantID(c.Request.Context())

	project, err := h.service.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	tenantID := appContext.MustTenantID(c.Request.Context())

	if err := h.service.Delete(c.Request.Context(), id, tenantID); err != nil {
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}
