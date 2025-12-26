package testutil

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Tenant represents a test tenant fixture
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// User represents a test user fixture
type User struct {
	ID                 string
	Auth0ID            string
	Email              string
	Firstname          string
	Lastname           string
	LastActiveTenantID *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Project represents a test project fixture
type Project struct {
	ID           string
	TenantID     string
	Name         string
	ClientAPIKey string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TenantMember represents a test tenant membership fixture
type TenantMember struct {
	ID        string
	UserID    string
	TenantID  string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Flag represents a test flag fixture
type Flag struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Enabled     bool
	Rules       string
	RuleLogic   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateTenant creates a tenant in the database for testing
func CreateTenant(t *testing.T, tx *sqlx.Tx, name, slug string) *Tenant {
	t.Helper()

	tenant := &Tenant{
		ID:        uuid.New().String(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO tenants (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := tx.Exec(query, tenant.ID, tenant.Name, tenant.Slug, tenant.CreatedAt, tenant.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	return tenant
}

// CreateUser creates a user in the database for testing
func CreateUser(t *testing.T, tx *sqlx.Tx, auth0ID, email, firstname, lastname string) *User {
	t.Helper()

	user := &User{
		ID:        uuid.New().String(),
		Auth0ID:   auth0ID,
		Email:     email,
		Firstname: firstname,
		Lastname:  lastname,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO users (id, auth0_id, email, firstname, lastname, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := tx.Exec(query, user.ID, user.Auth0ID, user.Email, user.Firstname, user.Lastname, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

// CreateProject creates a project in the database for testing
func CreateProject(t *testing.T, tx *sqlx.Tx, tenantID, name, apiKey string) *Project {
	t.Helper()

	project := &Project{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         name,
		ClientAPIKey: apiKey,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO projects (id, tenant_id, name, client_api_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.Exec(query, project.ID, project.TenantID, project.Name, project.ClientAPIKey, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	return project
}

// CreateTenantMember creates a tenant membership in the database for testing
func CreateTenantMember(t *testing.T, tx *sqlx.Tx, userID, tenantID, role string) *TenantMember {
	t.Helper()

	member := &TenantMember{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Time{},
	}

	query := `
		INSERT INTO tenant_members (id, user_id, tenant_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.Exec(query, member.ID, member.UserID, member.TenantID, member.Role, member.CreatedAt, member.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create tenant member: %v", err)
	}

	return member
}

// SetUserLastActiveTenant sets the last_active_tenant_id for a user
func SetUserLastActiveTenant(t *testing.T, tx *sqlx.Tx, userID, tenantID string) {
	t.Helper()

	query := `UPDATE users SET last_active_tenant_id = $1, updated_at = $2 WHERE id = $3`
	_, err := tx.Exec(query, tenantID, time.Now(), userID)
	if err != nil {
		t.Fatalf("failed to set last active tenant: %v", err)
	}
}

// CreateFlag creates a feature flag in the database for testing
func CreateFlag(t *testing.T, tx *sqlx.Tx, projectID, name, description string, enabled bool) *Flag {
	t.Helper()

	flag := &Flag{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Enabled:     enabled,
		Rules:       "[]",
		RuleLogic:   "AND",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO flags (id, project_id, name, description, enabled, rules, rule_logic, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.Exec(query, flag.ID, flag.ProjectID, flag.Name, flag.Description, flag.Enabled, flag.Rules, flag.RuleLogic, flag.CreatedAt, flag.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}

	return flag
}

// CreateFlagWithRules creates a feature flag with custom rules and rule logic
func CreateFlagWithRules(t *testing.T, tx *sqlx.Tx, projectID, name, description string, enabled bool, rules string, ruleLogic string) *Flag {
	t.Helper()

	flag := &Flag{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Enabled:     enabled,
		Rules:       rules,
		RuleLogic:   ruleLogic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO flags (id, project_id, name, description, enabled, rules, rule_logic, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.Exec(query, flag.ID, flag.ProjectID, flag.Name, flag.Description, flag.Enabled, flag.Rules, flag.RuleLogic, flag.CreatedAt, flag.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}

	return flag
}
