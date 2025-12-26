package evaluation

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
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
	r.POST("/evaluate", h.EvaluateAll)
	r.POST("/flags/:id/evaluate", h.EvaluateSingle)
}

// EvaluateAll handles bulk evaluation for all flags in a project
func (h *handler) EvaluateAll(c *gin.Context) {
	var req EvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract project_id from context (set by API key middleware)
	projectID := appContext.MustProjectID(c.Request.Context())

	result, err := h.service.EvaluateAll(c.Request.Context(), projectID, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "evaluation failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// EvaluateSingle handles evaluation for a single flag
func (h *handler) EvaluateSingle(c *gin.Context) {
	flagID := c.Param("id")

	var req SingleEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract tenant_id from context (set by API key middleware)
	tenantID := appContext.MustTenantID(c.Request.Context())

	result, err := h.service.EvaluateSingle(c.Request.Context(), flagID, tenantID, req.Context)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}
