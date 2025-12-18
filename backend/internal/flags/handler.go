package flag

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
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

	flag := &Flag{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     false,
		Rules:       req.Rules,
	}

	if flag.Rules == nil {
		flag.Rules = []Rule{}
	}

	if err := h.service.Create(flag); err != nil {
		if errors.Is(err, ErrInvalidFlagData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create flag"})
		return
	}

	c.JSON(http.StatusCreated, flag)
}

func (h *handler) List(c *gin.Context) {
	flags, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list flags"})
		return
	}

	c.JSON(http.StatusOK, flags)
}

func (h *handler) Get(c *gin.Context) {
	id := c.Param("id")

	flag, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, ErrFlagNotFound) {
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

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flag, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, ErrFlagNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get flag"})
		return
	}

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

	if err := h.service.Update(flag); err != nil {
		if errors.Is(err, ErrInvalidFlagData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrFlagNotFound) {
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

	flag, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, ErrFlagNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get flag"})
		return
	}

	flag.Enabled = !flag.Enabled

	if err := h.service.Update(flag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle flag"})
		return
	}

	c.JSON(http.StatusOK, flag)
}

func (h *handler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, ErrFlagNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete flag"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
