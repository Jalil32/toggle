package flag

import (
	"database/sql"
	"errors"
	"testing"
)

type mockRepository struct {
	createFunc  func(f *Flag) error
	getByIDFunc func(id string) (*Flag, error)
	listFunc    func() ([]Flag, error)
	updateFunc  func(f *Flag) error
	deleteFunc  func(id string) error
}

func (m *mockRepository) Create(f *Flag) error {
	if m.createFunc != nil {
		return m.createFunc(f)
	}
	return nil
}

func (m *mockRepository) GetByID(id string) (*Flag, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return nil, nil
}

func (m *mockRepository) List() ([]Flag, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return nil, nil
}

func (m *mockRepository) Update(f *Flag) error {
	if m.updateFunc != nil {
		return m.updateFunc(f)
	}
	return nil
}

func (m *mockRepository) Delete(id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func TestServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		flag    *Flag
		mockFn  func(f *Flag) error
		wantErr error
	}{
		{
			name: "successful creation",
			flag: &Flag{
				Name:        "test-flag",
				Description: "test description",
				Rules:       []Rule{},
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
			},
			mockFn:  nil,
			wantErr: ErrInvalidFlagData,
		},
		{
			name: "repository error",
			flag: &Flag{
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(f *Flag) error {
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
			svc := NewService(mockRepo)

			err := svc.Create(tt.flag)

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
		mockFn  func(id string) (*Flag, error)
		want    *Flag
		wantErr error
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
			mockFn: func(id string) (*Flag, error) {
				return nil, sql.ErrNoRows
			},
			want:    nil,
			wantErr: ErrFlagNotFound,
		},
		{
			name: "repository error",
			id:   "test-id",
			mockFn: func(id string) (*Flag, error) {
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
			svc := NewService(mockRepo)

			flag, err := svc.GetByID(tt.id)

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
		mockFn  func() ([]Flag, error)
		want    []Flag
		wantErr error
	}{
		{
			name: "successful list",
			mockFn: func() ([]Flag, error) {
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
			mockFn: func() ([]Flag, error) {
				return nil, nil
			},
			want:    []Flag{},
			wantErr: nil,
		},
		{
			name: "repository error",
			mockFn: func() ([]Flag, error) {
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
			svc := NewService(mockRepo)

			flags, err := svc.List()

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
		mockFn  func(f *Flag) error
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
			mockFn: func(f *Flag) error {
				return sql.ErrNoRows
			},
			wantErr: ErrFlagNotFound,
		},
		{
			name: "repository error",
			flag: &Flag{
				ID:          "test-id",
				Name:        "test-flag",
				Description: "test description",
			},
			mockFn: func(f *Flag) error {
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
			svc := NewService(mockRepo)

			err := svc.Update(tt.flag)

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
		mockFn  func(id string) error
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
			mockFn: func(id string) error {
				return sql.ErrNoRows
			},
			wantErr: ErrFlagNotFound,
		},
		{
			name: "repository error",
			id:   "test-id",
			mockFn: func(id string) error {
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
			svc := NewService(mockRepo)

			err := svc.Delete(tt.id)

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
