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
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   "test-project-id",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow("generated-uuid", time.Now(), time.Now())
				mock.ExpectQuery("INSERT INTO flags").
					WithArgs(
						"test-flag",
						"test description",
						false,
						sqlmock.AnyArg(),
						"AND",
						"test-project-id",
					).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "database error",
			flag: &Flag{
				Name:        "test-flag",
				Description: "test description",
				Enabled:     false,
				Rules:       []Rule{},
				RuleLogic:   "AND",
				ProjectID:   "test-project-id",
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
				rows := sqlmock.NewRows([]string{"id", "name", "description", "enabled", "rules", "rule_logic", "project_id", "created_at", "updated_at"}).
					AddRow("test-id", "test-flag", "test description", false, rulesJSON, "AND", "test-project-id", now, now)
				mock.ExpectQuery("SELECT f.id, f.name, f.description, f.enabled, f.rules, f.rule_logic, f.project_id, f.created_at, f.updated_at FROM flags f INNER JOIN projects p ON f.project_id = p.id WHERE f.id = \\$1 AND p.tenant_id = \\$2").
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
				mock.ExpectQuery("SELECT f.id, f.name, f.description, f.enabled, f.rules, f.rule_logic, f.project_id, f.created_at, f.updated_at FROM flags f INNER JOIN projects p ON f.project_id = p.id WHERE f.id = \\$1 AND p.tenant_id = \\$2").
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
				mock.ExpectQuery("SELECT f.id, f.name, f.description, f.enabled, f.rules, f.rule_logic, f.project_id, f.created_at, f.updated_at FROM flags f INNER JOIN projects p ON f.project_id = p.id WHERE f.id = \\$1 AND p.tenant_id = \\$2").
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
				rows := sqlmock.NewRows([]string{"id", "name", "description", "enabled", "rules", "rule_logic", "project_id", "created_at", "updated_at"}).
					AddRow("id1", "flag1", "desc1", true, rulesJSON, "AND", "test-project-id", now, now).
					AddRow("id2", "flag2", "desc2", false, rulesJSON, "AND", "test-project-id", now, now)
				mock.ExpectQuery("SELECT f.id, f.name, f.description, f.enabled, f.rules, f.rule_logic, f.project_id, f.created_at, f.updated_at FROM flags f INNER JOIN projects p ON f.project_id = p.id WHERE p.tenant_id = \\$1 ORDER BY f.created_at DESC").
					WithArgs("test-tenant-id").
					WillReturnRows(rows)
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "empty list",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "description", "enabled", "rules", "rule_logic", "project_id", "created_at", "updated_at"})
				mock.ExpectQuery("SELECT f.id, f.name, f.description, f.enabled, f.rules, f.rule_logic, f.project_id, f.created_at, f.updated_at FROM flags f INNER JOIN projects p ON f.project_id = p.id WHERE p.tenant_id = \\$1 ORDER BY f.created_at DESC").
					WithArgs("test-tenant-id").
					WillReturnRows(rows)
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectQuery("SELECT f.id, f.name, f.description, f.enabled, f.rules, f.rule_logic, f.project_id, f.created_at, f.updated_at FROM flags f INNER JOIN projects p ON f.project_id = p.id WHERE p.tenant_id = \\$1 ORDER BY f.created_at DESC").
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
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE flags f SET name = \\$2, description = \\$3, enabled = \\$4, rules = \\$5, rule_logic = \\$6, updated_at = \\$7 FROM projects p WHERE f.id = \\$1 AND f.project_id = p.id AND p.tenant_id = \\$8").
					WithArgs(
						"test-id",
						"updated-flag",
						"updated description",
						true,
						sqlmock.AnyArg(),
						"AND",
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
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE flags f SET name = \\$2, description = \\$3, enabled = \\$4, rules = \\$5, rule_logic = \\$6, updated_at = \\$7 FROM projects p WHERE f.id = \\$1 AND f.project_id = p.id AND p.tenant_id = \\$8").
					WithArgs(
						"non-existent",
						"test-flag",
						"test description",
						false,
						sqlmock.AnyArg(),
						"AND",
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
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE flags f SET name = \\$2, description = \\$3, enabled = \\$4, rules = \\$5, rule_logic = \\$6, updated_at = \\$7 FROM projects p WHERE f.id = \\$1 AND f.project_id = p.id AND p.tenant_id = \\$8").
					WithArgs(
						"test-id",
						"test-flag",
						"test description",
						false,
						sqlmock.AnyArg(),
						"AND",
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
				mock.ExpectExec("DELETE FROM flags f USING projects p WHERE f.id = \\$1 AND f.project_id = p.id AND p.tenant_id = \\$2").
					WithArgs("test-id", "test-tenant-id").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "non-existent",
			mockFn: func() {
				mock.ExpectExec("DELETE FROM flags f USING projects p WHERE f.id = \\$1 AND f.project_id = p.id AND p.tenant_id = \\$2").
					WithArgs("non-existent", "test-tenant-id").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
		{
			name: "database error",
			id:   "test-id",
			mockFn: func() {
				mock.ExpectExec("DELETE FROM flags f USING projects p WHERE f.id = \\$1 AND f.project_id = p.id AND p.tenant_id = \\$2").
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
