package flag

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
)

type Handler interface {
	RegisterRoutes(r *gin.RouterGroup)
}

type handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return &handler{service: service}
}

func (h *handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/flags", h.Create)
	r.GET("/flags", h.List)
	r.GET("/flags/:id", h.Get)
	r.PUT("/flags/:id", h.Update)
	r.PATCH("/flags/:id/toggle", h.Toggle)
	r.DELETE("/flags/:id", h.Delete)
}

func (h *handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := appContext.MustTenantID(c.Request.Context())

	flag := &Flag{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		Enabled:     false,
		Rules:       req.Rules,
	}

	if flag.Rules == nil {
		flag.Rules = []Rule{}
	}

	if err := h.service.Create(c.Request.Context(), flag, tenantID); err != nil {
		if errors.Is(err, ErrInvalidFlagData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create flag"})
		return
	}

	c.JSON(http.StatusCreated, flag)
}

func (h *handler) List(c *gin.Context) {
	tenantID := appContext.MustTenantID(c.Request.Context())

	flags, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list flags"})
		return
	}

	c.JSON(http.StatusOK, flags)
}

func (h *handler) Get(c *gin.Context) {
	id := c.Param("id")
	tenantID := appContext.MustTenantID(c.Request.Context())

	flag, err := h.service.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get flag"})
		return
	}

	c.JSON(http.StatusOK, flag)
}

func (h *handler) Update(c *gin.Context) {
	id := c.Param("id")
	tenantID := appContext.MustTenantID(c.Request.Context())

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch existing flag first to ensure it exists and belongs to tenant
	flag, err := h.service.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get flag"})
		return
	}

	// Apply updates
	if req.Name != nil {
		flag.Name = *req.Name
	}
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.Enabled != nil {
		flag.Enabled = *req.Enabled
	}
	if req.Rules != nil {
		flag.Rules = req.Rules
	}

	if err := h.service.Update(c.Request.Context(), flag, tenantID); err != nil {
		if errors.Is(err, ErrInvalidFlagData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update flag"})
		return
	}

	c.JSON(http.StatusOK, flag)
}

func (h *handler) Toggle(c *gin.Context) {
	id := c.Param("id")
	tenantID := appContext.MustTenantID(c.Request.Context())

	flag, err := h.service.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get flag"})
		return
	}

	flag.Enabled = !flag.Enabled

	if err := h.service.Update(c.Request.Context(), flag, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle flag"})
		return
	}

	c.JSON(http.StatusOK, flag)
}

func (h *handler) Delete(c *gin.Context) {
	id := c.Param("id")
	tenantID := appContext.MustTenantID(c.Request.Context())

	if err := h.service.Delete(c.Request.Context(), id, tenantID); err != nil {
		if pkgErrors.IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete flag"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
