package flag

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockService struct {
	createFunc  func(f *Flag) error
	getByIDFunc func(id string) (*Flag, error)
	listFunc    func() ([]Flag, error)
	updateFunc  func(f *Flag) error
	deleteFunc  func(id string) error
}

func (m *mockService) Create(f *Flag) error {
	if m.createFunc != nil {
		return m.createFunc(f)
	}
	return nil
}

func (m *mockService) GetByID(id string) (*Flag, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return nil, nil
}

func (m *mockService) List() ([]Flag, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return nil, nil
}

func (m *mockService) Update(f *Flag) error {
	if m.updateFunc != nil {
		return m.updateFunc(f)
	}
	return nil
}

func (m *mockService) Delete(id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandlerCreate(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockFn         func(f *Flag) error
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful creation",
			body: CreateRequest{
				Name:        "test-flag",
				Description: "test description",
				Rules:       []Rule{},
			},
			mockFn: func(f *Flag) error {
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
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(f *Flag) error {
				return ErrInvalidFlagData
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "service error",
			body: CreateRequest{
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(f *Flag) error {
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
			req := httptest.NewRequest(http.MethodPost, "/flags", bytes.NewReader(bodyBytes))
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
		mockFn         func() ([]Flag, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful list",
			mockFn: func() ([]Flag, error) {
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
			mockFn: func() ([]Flag, error) {
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
			mockFn: func() ([]Flag, error) {
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

			req := httptest.NewRequest(http.MethodGet, "/flags", nil)
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
		mockFn         func(id string) (*Flag, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful get",
			id:   "test-id",
			mockFn: func(id string) (*Flag, error) {
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
			mockFn: func(id string) (*Flag, error) {
				return nil, ErrFlagNotFound
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name: "service error",
			id:   "test-id",
			mockFn: func(id string) (*Flag, error) {
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

			req := httptest.NewRequest(http.MethodGet, "/flags/"+tt.id, nil)
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
		mockGetFn      func(id string) (*Flag, error)
		mockUpdateFn   func(f *Flag) error
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
			mockGetFn: func(id string) (*Flag, error) {
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
			mockGetFn: func(id string) (*Flag, error) {
				return nil, ErrFlagNotFound
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
			mockGetFn: func(id string) (*Flag, error) {
				return &Flag{
					ID:   id,
					Name: "old-name",
				}, nil
			},
			mockUpdateFn: func(f *Flag) error {
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
			req := httptest.NewRequest(http.MethodPut, "/flags/"+tt.id, bytes.NewReader(bodyBytes))
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
		mockGetFn      func(id string) (*Flag, error)
		mockUpdateFn   func(f *Flag) error
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful toggle from false to true",
			id:   "test-id",
			mockGetFn: func(id string) (*Flag, error) {
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
			mockGetFn: func(id string) (*Flag, error) {
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
			mockGetFn: func(id string) (*Flag, error) {
				return nil, ErrFlagNotFound
			},
			mockUpdateFn:   nil,
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name: "update error",
			id:   "test-id",
			mockGetFn: func(id string) (*Flag, error) {
				return &Flag{
					ID:      id,
					Name:    "test-flag",
					Enabled: false,
				}, nil
			},
			mockUpdateFn: func(f *Flag) error {
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

			req := httptest.NewRequest(http.MethodPatch, "/flags/"+tt.id+"/toggle", nil)
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
		mockFn         func(id string) error
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
			mockFn: func(id string) error {
				return ErrFlagNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "service error",
			id:   "test-id",
			mockFn: func(id string) error {
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

			req := httptest.NewRequest(http.MethodDelete, "/flags/"+tt.id, nil)
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
