package flag

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(f *Flag) error
	GetByID(id string) (*Flag, error)
	List() ([]Flag, error)
	Update(f *Flag) error
	Delete(id string) error
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(f *Flag) error {
	rulesJSON, err := json.Marshal(f.Rules)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO flags (name, description, enabled, rules, project_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err = r.db.QueryRow(query, f.Name, f.Description, f.Enabled, rulesJSON, f.ProjectID).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *postgresRepository) GetByID(id string) (*Flag, error) {
	var f Flag
	var rulesJSON []byte

	query := `
		SELECT id, name, description, enabled, rules, created_at, updated_at
		FROM flags
		WHERE id = $1
	`

	err := r.db.QueryRowx(query, id).Scan(
		&f.ID, &f.Name, &f.Description, &f.Enabled, &rulesJSON, &f.CreatedAt, &f.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rulesJSON, &f.Rules); err != nil {
		return nil, err
	}

	return &f, nil
}

func (r *postgresRepository) List() ([]Flag, error) {
	query := `
		SELECT id, name, description, enabled, rules, created_at, updated_at
		FROM flags
		ORDER BY created_at DESC
	`
	rows, err := r.db.Queryx(query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var flags []Flag

	for rows.Next() {
		var f Flag
		var rulesJSON []byte

		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Enabled, &rulesJSON, &f.CreatedAt, &f.UpdatedAt)
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

func (r *postgresRepository) Update(f *Flag) error {
	rulesJSON, err := json.Marshal(f.Rules)
	if err != nil {
		return err
	}

	now := time.Now()
	query := `
		UPDATE flags
		SET name = $2, description = $3, enabled = $4, rules = $5, updated_at = $6
		WHERE id = $1
	`
	result, err := r.db.Exec(query, f.ID, f.Name, f.Description, f.Enabled, rulesJSON, now)
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

func (r *postgresRepository) Delete(id string) error {
	query := `DELETE FROM flags WHERE id = $1`
	result, err := r.db.Exec(query, id)
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
