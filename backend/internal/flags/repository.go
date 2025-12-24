package flag

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, f *Flag) error
	GetByID(ctx context.Context, id string, tenantID string) (*Flag, error)
	List(ctx context.Context, tenantID string) ([]Flag, error)
	ListByProject(ctx context.Context, projectID string, tenantID string) ([]Flag, error)
	Update(ctx context.Context, f *Flag, tenantID string) error
	Delete(ctx context.Context, id string, tenantID string) error
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

// DBContext is an interface that both *sqlx.DB and *sqlx.Tx implement
type DBContext interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
}

// getDB returns the transaction from context if present, otherwise returns the DB
func (r *postgresRepository) getDB(ctx context.Context) DBContext {
	if tx, ok := transaction.GetTx(ctx); ok {
		return tx
	}
	return r.db
}

func (r *postgresRepository) Create(ctx context.Context, f *Flag) error {
	rulesJSON, err := json.Marshal(f.Rules)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO flags (name, description, enabled, rules, project_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err = r.getDB(ctx).QueryRowContext(ctx, query, f.Name, f.Description, f.Enabled, rulesJSON, f.ProjectID).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string, tenantID string) (*Flag, error) {
	var f Flag
	var rulesJSON []byte

	// Join with projects to enforce tenant boundary
	query := `
		SELECT f.id, f.name, f.description, f.enabled, f.rules, f.project_id,
		       f.created_at, f.updated_at
		FROM flags f
		INNER JOIN projects p ON f.project_id = p.id
		WHERE f.id = $1 AND p.tenant_id = $2
	`

	err := r.getDB(ctx).QueryRowxContext(ctx, query, id, tenantID).Scan(
		&f.ID, &f.Name, &f.Description, &f.Enabled, &rulesJSON, &f.ProjectID,
		&f.CreatedAt, &f.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rulesJSON, &f.Rules); err != nil {
		return nil, err
	}

	return &f, nil
}

func (r *postgresRepository) List(ctx context.Context, tenantID string) ([]Flag, error) {
	// Join with projects to enforce tenant boundary
	query := `
		SELECT f.id, f.name, f.description, f.enabled, f.rules, f.project_id,
		       f.created_at, f.updated_at
		FROM flags f
		INNER JOIN projects p ON f.project_id = p.id
		WHERE p.tenant_id = $1
		ORDER BY f.created_at DESC
	`
	rows, err := r.getDB(ctx).QueryxContext(ctx, query, tenantID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var flags []Flag

	for rows.Next() {
		var f Flag
		var rulesJSON []byte

		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Enabled, &rulesJSON, &f.ProjectID,
			&f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(rulesJSON, &f.Rules); err != nil {
			return nil, err
		}

		flags = append(flags, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return flags, nil
}

// ListByProject returns all flags for a specific project within a tenant
func (r *postgresRepository) ListByProject(ctx context.Context, projectID string, tenantID string) ([]Flag, error) {
	query := `
		SELECT f.id, f.name, f.description, f.enabled, f.rules, f.project_id,
		       f.created_at, f.updated_at
		FROM flags f
		INNER JOIN projects p ON f.project_id = p.id
		WHERE f.project_id = $1 AND p.tenant_id = $2
		ORDER BY f.created_at DESC
	`
	rows, err := r.getDB(ctx).QueryxContext(ctx, query, projectID, tenantID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var flags []Flag

	for rows.Next() {
		var f Flag
		var rulesJSON []byte

		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Enabled, &rulesJSON, &f.ProjectID,
			&f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(rulesJSON, &f.Rules); err != nil {
			return nil, err
		}

		flags = append(flags, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return flags, nil
}

func (r *postgresRepository) Update(ctx context.Context, f *Flag, tenantID string) error {
	rulesJSON, err := json.Marshal(f.Rules)
	if err != nil {
		return err
	}

	now := time.Now()

	// Verify tenant ownership via project join
	query := `
		UPDATE flags f
		SET name = $2, description = $3, enabled = $4, rules = $5, updated_at = $6
		FROM projects p
		WHERE f.id = $1
		  AND f.project_id = p.id
		  AND p.tenant_id = $7
	`
	result, err := r.getDB(ctx).ExecContext(ctx, query,
		f.ID, f.Name, f.Description, f.Enabled, rulesJSON, now, tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	f.UpdatedAt = now
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id string, tenantID string) error {
	// Verify tenant ownership via project join
	query := `
		DELETE FROM flags f
		USING projects p
		WHERE f.id = $1
		  AND f.project_id = p.id
		  AND p.tenant_id = $2
	`
	result, err := r.getDB(ctx).ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
