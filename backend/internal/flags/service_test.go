package flag

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
)

type mockValidator struct {
	validateProjectOwnershipFunc func(ctx context.Context, projectID, tenantID string) error
	validateTenantExistsFunc     func(ctx context.Context, tenantID string) error
}

func (m *mockValidator) ValidateProjectOwnership(ctx context.Context, projectID, tenantID string) error {
	if m.validateProjectOwnershipFunc != nil {
		return m.validateProjectOwnershipFunc(ctx, projectID, tenantID)
	}
	return nil
}

func (m *mockValidator) ValidateTenantExists(ctx context.Context, tenantID string) error {
	if m.validateTenantExistsFunc != nil {
		return m.validateTenantExistsFunc(ctx, tenantID)
	}
	return nil
}

type mockRepository struct {
	createFunc      func(ctx context.Context, f *Flag) error
	getByIDFunc     func(ctx context.Context, id string, tenantID string) (*Flag, error)
	listFunc        func(ctx context.Context, tenantID string) ([]Flag, error)
	listByProjectFn func(ctx context.Context, projectID string, tenantID string) ([]Flag, error)
	updateFunc      func(ctx context.Context, f *Flag, tenantID string) error
	deleteFunc      func(ctx context.Context, id string, tenantID string) error
}

func (m *mockRepository) Create(ctx context.Context, f *Flag) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, f)
	}
	// Set a test ID to simulate database behavior
	if f != nil && f.ID == "" {
		f.ID = "test-generated-id"
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string, tenantID string) (*Flag, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, tenantID)
	}
	return nil, nil
}

func (m *mockRepository) List(ctx context.Context, tenantID string) ([]Flag, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockRepository) ListByProject(ctx context.Context, projectID string, tenantID string) ([]Flag, error) {
	if m.listByProjectFn != nil {
		return m.listByProjectFn(ctx, projectID, tenantID)
	}
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, f *Flag, tenantID string) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, f, tenantID)
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string, tenantID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, tenantID)
	}
	return nil
}

// Note: We pass nil for validator in tests since validator logic is tested separately
// In production, actual validator is injected via dependency injection

func TestServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		flag    *Flag
		mockFn  func(ctx context.Context, f *Flag) error
		wantErr error
	}{
		{
			name: "successful creation",
			flag: &Flag{
				Name:        "test-flag",
				Description: "test description",
				Rules:       []Rule{},
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn:  nil,
			wantErr: nil,
		},
		{
			name:    "nil flag",
			flag:    nil,
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "empty name",
			flag: &Flag{
				Name:        "",
				Description: "test description",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "repository error",
			flag: &Flag{
				Name:        "test-flag",
				Description: "test description",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn: func(ctx context.Context, f *Flag) error {
				return errors.New("database error")
			},
			wantErr: errors.New("failed to create flag: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{
				createFunc: tt.mockFn,
			}
			mockVal := &mockValidator{}
			svc := NewService(mockRepo, mockVal, slog.Default())

			err := svc.Create(context.Background(), tt.flag, "test-tenant-id")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error containing %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if tt.flag != nil && tt.flag.ID == "" {
					t.Error("expected flag ID to be set")
				}
			}
		})
	}
}

func TestServiceGetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		mockFn  func(ctx context.Context, id string, tenantID string) (*Flag, error)
		want    *Flag
		wantErr error
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
			want: &Flag{
				ID:          "test-id",
				Name:        "test-flag",
				Description: "test description",
			},
			wantErr: nil,
		},
		{
			name:    "empty id",
			id:      "",
			mockFn:  nil,
			want:    nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "flag not found",
			id:   "non-existent",
			mockFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return nil, sql.ErrNoRows
			},
			want:    nil,
			wantErr: pkgErrors.ErrNotFound,
		},
		{
			name: "repository error",
			id:   "test-id",
			mockFn: func(ctx context.Context, id string, tenantID string) (*Flag, error) {
				return nil, errors.New("database error")
			},
			want:    nil,
			wantErr: errors.New("failed to get flag: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{
				getByIDFunc: tt.mockFn,
			}
			mockVal := &mockValidator{}
			svc := NewService(mockRepo, mockVal, slog.Default())

			flag, err := svc.GetByID(context.Background(), tt.id, "test-tenant-id")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if flag == nil {
					t.Error("expected flag to be returned")
				} else if flag.ID != tt.want.ID || flag.Name != tt.want.Name {
					t.Errorf("expected flag %+v, got %+v", tt.want, flag)
				}
			}
		})
	}
}

func TestServiceList(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(ctx context.Context, tenantID string) ([]Flag, error)
		want    []Flag
		wantErr error
	}{
		{
			name: "successful list",
			mockFn: func(ctx context.Context, tenantID string) ([]Flag, error) {
				return []Flag{
					{ID: "1", Name: "flag1"},
					{ID: "2", Name: "flag2"},
				}, nil
			},
			want:    []Flag{{ID: "1", Name: "flag1"}, {ID: "2", Name: "flag2"}},
			wantErr: nil,
		},
		{
			name: "empty list",
			mockFn: func(ctx context.Context, tenantID string) ([]Flag, error) {
				return nil, nil
			},
			want:    []Flag{},
			wantErr: nil,
		},
		{
			name: "repository error",
			mockFn: func(ctx context.Context, tenantID string) ([]Flag, error) {
				return nil, errors.New("database error")
			},
			want:    nil,
			wantErr: errors.New("failed to list flags: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{
				listFunc: tt.mockFn,
			}
			mockVal := &mockValidator{}
			svc := NewService(mockRepo, mockVal, slog.Default())

			flags, err := svc.List(context.Background(), "test-tenant-id")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if len(flags) != len(tt.want) {
					t.Errorf("expected %d flags, got %d", len(tt.want), len(flags))
				}
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	tests := []struct {
		name    string
		flag    *Flag
		mockFn  func(ctx context.Context, f *Flag, tenantID string) error
		wantErr error
	}{
		{
			name: "successful update",
			flag: &Flag{
				ID:          "test-id",
				Name:        "updated-flag",
				Description: "updated description",
			},
			mockFn:  nil,
			wantErr: nil,
		},
		{
			name:    "nil flag",
			flag:    nil,
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "empty name",
			flag: &Flag{
				ID:          "test-id",
				Name:        "",
				Description: "test description",
			},
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "empty id",
			flag: &Flag{
				ID:          "",
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "flag not found",
			flag: &Flag{
				ID:          "non-existent",
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(ctx context.Context, f *Flag, tenantID string) error {
				return sql.ErrNoRows
			},
			wantErr: pkgErrors.ErrNotFound,
		},
		{
			name: "repository error",
			flag: &Flag{
				ID:          "test-id",
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(ctx context.Context, f *Flag, tenantID string) error {
				return errors.New("database error")
			},
			wantErr: errors.New("failed to update flag: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{
				updateFunc: tt.mockFn,
			}
			mockVal := &mockValidator{}
			svc := NewService(mockRepo, mockVal, slog.Default())

			err := svc.Update(context.Background(), tt.flag, "test-tenant-id")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestServiceDelete(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		mockFn  func(ctx context.Context, id string, tenantID string) error
		wantErr error
	}{
		{
			name:    "successful delete",
			id:      "test-id",
			mockFn:  nil,
			wantErr: nil,
		},
		{
			name:    "empty id",
			id:      "",
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "flag not found",
			id:   "non-existent",
			mockFn: func(ctx context.Context, id string, tenantID string) error {
				return sql.ErrNoRows
			},
			wantErr: pkgErrors.ErrNotFound,
		},
		{
			name: "repository error",
			id:   "test-id",
			mockFn: func(ctx context.Context, id string, tenantID string) error {
				return errors.New("database error")
			},
			wantErr: errors.New("failed to delete flag: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{
				deleteFunc: tt.mockFn,
			}
			mockVal := &mockValidator{}
			svc := NewService(mockRepo, mockVal, slog.Default())

			err := svc.Delete(context.Background(), tt.id, "test-tenant-id")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestValidateFlag(t *testing.T) {
	tests := []struct {
		name    string
		flag    *Flag
		wantErr error
	}{
		{
			name: "valid flag",
			flag: &Flag{
				Name:        "test-flag",
				Description: "test description",
			},
			wantErr: nil,
		},
		{
			name:    "nil flag",
			flag:    nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "empty name",
			flag: &Flag{
				Name:        "",
				Description: "test description",
			},
			wantErr: ErrInvalidFlagData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service{}
			err := svc.validateFlag(tt.flag)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}
