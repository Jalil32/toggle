package flag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
)

type mockService struct {
	createFunc  func(ctx context.Context, f *Flag, tenantID string) error
	getByIDFunc func(ctx context.Context, id string, tenantID string) (*Flag, error)
	listFunc    func(ctx context.Context, tenantID string) ([]Flag, error)
	updateFunc  func(ctx context.Context, f *Flag, tenantID string) error
	deleteFunc  func(ctx context.Context, id string, tenantID string) error
}

func (m *mockService) Create(ctx context.Context, f *Flag, tenantID string) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, f, tenantID)
	}
	return nil
}

func (m *mockService) GetByID(ctx context.Context, id string, tenantID string) (*Flag, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, tenantID)
	}
	return nil, nil
}

func (m *mockService) List(ctx context.Context, tenantID string) ([]Flag, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockService) Update(ctx context.Context, f *Flag, tenantID string) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, f, tenantID)
	}
	return nil
}

func (m *mockService) Delete(ctx context.Context, id string, tenantID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, tenantID)
	}
	return nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// setupTestContext creates a test context with auth values
func setupTestContext(userID, tenantID, role, auth0ID string) context.Context {
	ctx := context.Background()
	return appContext.WithAuth(ctx, userID, tenantID, role, auth0ID)
}

func TestHandlerCreate(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockFn         func(ctx context.Context, f *Flag, tenantID string) error
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful creation",
			body: CreateRequest{
				ProjectID:   "test-project-id",
				Name:        "test-flag",
				Description: "test description",
				Rules:       []Rule{},
			},
			mockFn: func(ctx context.Context, f *Flag, tenantID string) error {
				f.ID = "generated-id"
				return nil
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var flag Flag
				if err := json.Unmarshal(body, &flag); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if flag.Name != "test-flag" {
					t.Errorf("expected name 'test-flag', got '%s'", flag.Name)
				}
				if flag.ID == "" {
					t.Error("expected ID to be set")
				}
			},
		},
		{
			name:           "invalid json",
			body:           "invalid",
			mockFn:         nil,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "empty name",
			body: CreateRequest{
				ProjectID:   "test-project-id",
				Name:        "",
				Description: "test description",
			},
			mockFn:         nil,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "service validation error",
			body: CreateRequest{
				ProjectID:   "test-project-id",
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(ctx context.Context, f *Flag, tenantID string) error {
				return ErrInvalidFlagData
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "service error",
			body: CreateRequest{
				ProjectID:   "test-project-id",
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(ctx context.Context, f *Flag, tenantID string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				createFunc: tt.mockFn,
			}
			h := NewHandler(mockSvc)

			router := setupTestRouter()
			router.POST("/flags", h.(*handler).Create)

			bodyBytes, _ := json.Marshal(tt.body)
			ctx := setupTestContext("test-user-id", "test-tenant-id", "admin", "test-auth0-id")
			req := httptest.NewRequest(http.MethodPost, "/flags", bytes.NewReader(bodyBytes))
			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestHandlerList(t *testing.T) {
	tests := []struct {
		name           string
		mockFn         func(ctx context.Context, tenantID string) ([]Flag, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful list",
			mockFn: func(ctx context.Context, tenantID string) ([]Flag, error) {
				return []Flag{
					{ID: "1", Name: "flag1", Description: "desc1"},
					{ID: "2", Name: "flag2", Description: "desc2"},
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var flags []Flag
				if err := json.Unmarshal(body, &flags); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if len(flags) != 2 {
					t.Errorf("expected 2 flags, got %d", len(flags))
				}
			},
		},
		{
			name: "empty list",
			mockFn: func(ctx context.Context, tenantID string) ([]Flag, error) {
				return []Flag{}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var flags []Flag
				if err := json.Unmarshal(body, &flags); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if len(flags) != 0 {
					t.Errorf("expected 0 flags, got %d", len(flags))
				}
			},
		},
		{
			name: "service error",
			mockFn: func(ctx context.Context, tenantID string) ([]Flag, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				listFunc: tt.mockFn,
			}
			h := NewHandler(mockSvc)

			router := setupTestRouter()
			router.GET("/flags", h.(*handler).List)

			ctx := setupTestContext("test-user-id", "test-tenant-id", "admin", "test-auth0-id")
			req := httptest.NewRequest(http.MethodGet, "/flags", nil)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestHandlerGet(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockFn         func(ctx context.Context, id string, tenantID string) (*Flag, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful get",
			id:   "test-id",
			mockFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return &Flag{
					ID:          id,
					Name:        "test-flag",
					Description: "test description",
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var flag Flag
				if err := json.Unmarshal(body, &flag); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if flag.ID != "test-id" || flag.Name != "test-flag" {
					t.Errorf("unexpected flag data: %+v", flag)
				}
			},
		},
		{
			name: "not found",
			id:   "non-existent",
			mockFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return nil, pkgErrors.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name: "service error",
			id:   "test-id",
			mockFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getByIDFunc: tt.mockFn,
			}
			h := NewHandler(mockSvc)

			router := setupTestRouter()
			router.GET("/flags/:id", h.(*handler).Get)

			ctx := setupTestContext("test-user-id", "test-tenant-id", "admin", "test-auth0-id")
			req := httptest.NewRequest(http.MethodGet, "/flags/"+tt.id, nil)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestHandlerUpdate(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		body           interface{}
		mockGetFn      func(ctx context.Context, id string, tenantID string) (*Flag, error)
		mockUpdateFn   func(ctx context.Context, f *Flag, tenantID string) error
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful update",
			id:   "test-id",
			body: UpdateRequest{
				Name:        stringPtr("updated-flag"),
				Description: stringPtr("updated description"),
			},
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return &Flag{
					ID:          id,
					Name:        "old-name",
					Description: "old description",
					Enabled:     false,
				}, nil
			},
			mockUpdateFn:   nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var flag Flag
				if err := json.Unmarshal(body, &flag); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if flag.Name != "updated-flag" {
					t.Errorf("expected name 'updated-flag', got '%s'", flag.Name)
				}
			},
		},
		{
			name:           "invalid json",
			id:             "test-id",
			body:           "invalid",
			mockGetFn:      nil,
			mockUpdateFn:   nil,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "flag not found",
			id:   "non-existent",
			body: UpdateRequest{
				Name: stringPtr("updated-flag"),
			},
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return nil, pkgErrors.ErrNotFound
			},
			mockUpdateFn:   nil,
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name: "update error",
			id:   "test-id",
			body: UpdateRequest{
				Name: stringPtr("updated-flag"),
			},
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return &Flag{
					ID:   id,
					Name: "old-name",
				}, nil
			},
			mockUpdateFn: func(ctx context.Context, f *Flag, tenantID string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getByIDFunc: tt.mockGetFn,
				updateFunc:  tt.mockUpdateFn,
			}
			h := NewHandler(mockSvc)

			router := setupTestRouter()
			router.PUT("/flags/:id", h.(*handler).Update)

			bodyBytes, _ := json.Marshal(tt.body)
			ctx := setupTestContext("test-user-id", "test-tenant-id", "admin", "test-auth0-id")
			req := httptest.NewRequest(http.MethodPut, "/flags/"+tt.id, bytes.NewReader(bodyBytes))
			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestHandlerToggle(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockGetFn      func(ctx context.Context, id string, tenantID string) (*Flag, error)
		mockUpdateFn   func(ctx context.Context, f *Flag, tenantID string) error
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful toggle from false to true",
			id:   "test-id",
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return &Flag{
					ID:      id,
					Name:    "test-flag",
					Enabled: false,
				}, nil
			},
			mockUpdateFn:   nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var flag Flag
				if err := json.Unmarshal(body, &flag); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if !flag.Enabled {
					t.Error("expected flag to be enabled")
				}
			},
		},
		{
			name: "successful toggle from true to false",
			id:   "test-id",
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return &Flag{
					ID:      id,
					Name:    "test-flag",
					Enabled: true,
				}, nil
			},
			mockUpdateFn:   nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var flag Flag
				if err := json.Unmarshal(body, &flag); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if flag.Enabled {
					t.Error("expected flag to be disabled")
				}
			},
		},
		{
			name: "flag not found",
			id:   "non-existent",
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return nil, pkgErrors.ErrNotFound
			},
			mockUpdateFn:   nil,
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name: "update error",
			id:   "test-id",
			mockGetFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return &Flag{
					ID:      id,
					Name:    "test-flag",
					Enabled: false,
				}, nil
			},
			mockUpdateFn: func(ctx context.Context, f *Flag, tenantID string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getByIDFunc: tt.mockGetFn,
				updateFunc:  tt.mockUpdateFn,
			}
			h := NewHandler(mockSvc)

			router := setupTestRouter()
			router.PATCH("/flags/:id/toggle", h.(*handler).Toggle)

			ctx := setupTestContext("test-user-id", "test-tenant-id", "admin", "test-auth0-id")
			req := httptest.NewRequest(http.MethodPatch, "/flags/"+tt.id+"/toggle", nil)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestHandlerDelete(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockFn         func(ctx context.Context, id string, tenantID string) error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			id:             "test-id",
			mockFn:         nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "not found",
			id:   "non-existent",
			mockFn: func(ctx context.Context, id string, tenantID string) error {
				return pkgErrors.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "service error",
			id:   "test-id",
			mockFn: func(ctx context.Context, id string, tenantID string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				deleteFunc: tt.mockFn,
			}
			h := NewHandler(mockSvc)

			router := setupTestRouter()
			router.DELETE("/flags/:id", h.(*handler).Delete)

			ctx := setupTestContext("test-user-id", "test-tenant-id", "admin", "test-auth0-id")
			req := httptest.NewRequest(http.MethodDelete, "/flags/"+tt.id, nil)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
