package flag

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestRepositoryCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	tests := []struct {
		name    string
		flag    *Flag
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful create",
			flag: &Flag{
				TenantID:    "test-tenant-id",
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow("generated-uuid", time.Now(), time.Now())
				mock.ExpectQuery("INSERT INTO flags").
					WithArgs(
						"test-tenant-id",
						stringPtr("test-project-id"),
						"test-flag",
						"test description",
						false,
						sqlmock.AnyArg(),
						"AND",
					).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "database error",
			flag: &Flag{
				TenantID:    "test-tenant-id",
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn: func() {
				mock.ExpectQuery("INSERT INTO flags").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.Create(context.Background(), tt.flag)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.wantErr {
				if tt.flag.ID == "" {
					t.Error("expected ID to be populated from database")
				}
				if tt.flag.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be populated from database")
				}
				if tt.flag.UpdatedAt.IsZero() {
					t.Error("expected UpdatedAt to be populated from database")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryGetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	now := time.Now()
	rulesJSON := []byte(`[]`)

	tests := []struct {
		name    string
		id      string
		mockFn  func()
		want    *Flag
		wantErr bool
	}{
		{
			name: "successful get",
			id:   "test-id",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "tenant_id", "project_id", "name", "description", "enabled", "rules", "rule_logic", "created_at", "updated_at"}).
					AddRow("test-id", "test-tenant-id", "test-project-id", "test-flag", "test description", false, rulesJSON, "AND", now, now)
				mock.ExpectQuery("SELECT id, tenant_id, project_id, name, description, enabled, rules, rule_logic").
					WithArgs("test-id", "test-tenant-id").
					WillReturnRows(rows)
			},
			want: &Flag{
				ID:          "test-id",
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "non-existent",
			mockFn: func() {
				mock.ExpectQuery("SELECT id, tenant_id, project_id, name, description, enabled, rules, rule_logic").
					WithArgs("non-existent", "test-tenant-id").
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "database error",
			id:   "test-id",
			mockFn: func() {
				mock.ExpectQuery("SELECT id, tenant_id, project_id, name, description, enabled, rules, rule_logic").
					WithArgs("test-id", "test-tenant-id").
					WillReturnError(sql.ErrConnDone)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			flag, err := repo.GetByID(context.Background(), tt.id, "test-tenant-id")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.wantErr && flag != nil {
				if flag.ID != tt.want.ID || flag.Name != tt.want.Name {
					t.Errorf("expected flag %+v, got %+v", tt.want, flag)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	now := time.Now()
	rulesJSON := []byte(`[]`)

	tests := []struct {
		name    string
		mockFn  func()
		want    int
		wantErr bool
	}{
		{
			name: "successful list",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "tenant_id", "project_id", "name", "description", "enabled", "rules", "rule_logic", "created_at", "updated_at"}).
					AddRow("id1", "test-tenant-id", "test-project-id", "flag1", "desc1", true, rulesJSON, "AND", now, now).
					AddRow("id2", "test-tenant-id", "test-project-id", "flag2", "desc2", false, rulesJSON, "AND", now, now)
				mock.ExpectQuery("SELECT id, tenant_id, project_id, name, description, enabled, rules, rule_logic").
					WithArgs("test-tenant-id").
					WillReturnRows(rows)
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "empty list",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "tenant_id", "project_id", "name", "description", "enabled", "rules", "rule_logic", "created_at", "updated_at"})
				mock.ExpectQuery("SELECT id, tenant_id, project_id, name, description, enabled, rules, rule_logic").
					WithArgs("test-tenant-id").
					WillReturnRows(rows)
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectQuery("SELECT id, tenant_id, project_id, name, description, enabled, rules, rule_logic").
					WithArgs("test-tenant-id").
					WillReturnError(sql.ErrConnDone)
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			flags, err := repo.List(context.Background(), "test-tenant-id")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.wantErr && len(flags) != tt.want {
				t.Errorf("expected %d flags, got %d", tt.want, len(flags))
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	tests := []struct {
		name    string
		flag    *Flag
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful update",
			flag: &Flag{
				ID:          "test-id",
				Name:        "updated-flag",
				Description: "updated description",
				Enabled:     true,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE flags").
					WithArgs(
						"test-id",
						"updated-flag",
						"updated description",
						true,
						sqlmock.AnyArg(),
						"AND",
						stringPtr("test-project-id"),
						sqlmock.AnyArg(),
						"test-tenant-id",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			flag: &Flag{
				ID:          "non-existent",
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE flags").
					WithArgs(
						"non-existent",
						"test-flag",
						"test description",
						false,
						sqlmock.AnyArg(),
						"AND",
						stringPtr("test-project-id"),
						sqlmock.AnyArg(),
						"test-tenant-id",
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
		{
			name: "database error",
			flag: &Flag{
				ID:          "test-id",
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   stringPtr("test-project-id"),
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE flags").
					WithArgs(
						"test-id",
						"test-flag",
						"test description",
						false,
						sqlmock.AnyArg(),
						"AND",
						stringPtr("test-project-id"),
						sqlmock.AnyArg(),
						"test-tenant-id",
					).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.Update(context.Background(), tt.flag, "test-tenant-id")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.wantErr && !tt.flag.UpdatedAt.IsZero() {
				if time.Since(tt.flag.UpdatedAt) > time.Second {
					t.Error("expected UpdatedAt to be set to current time")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	tests := []struct {
		name    string
		id      string
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   "test-id",
			mockFn: func() {
				mock.ExpectExec("DELETE FROM flags").
					WithArgs("test-id", "test-tenant-id").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "non-existent",
			mockFn: func() {
				mock.ExpectExec("DELETE FROM flags").
					WithArgs("non-existent", "test-tenant-id").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
		{
			name: "database error",
			id:   "test-id",
			mockFn: func() {
				mock.ExpectExec("DELETE FROM flags").
					WithArgs("test-id", "test-tenant-id").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.Delete(context.Background(), tt.id, "test-tenant-id")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}
