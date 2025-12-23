package tenants

import "time"

type Tenant struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TenantMember represents a user's membership in a tenant/workspace
type TenantMember struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	TenantID  string    `db:"tenant_id" json:"tenant_id"`
	Role      string    `db:"role" json:"role"` // owner, admin, member
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// TenantMembership represents a user's tenant membership with tenant details
type TenantMembership struct {
	TenantID   string `db:"tenant_id" json:"tenant_id"`
	Role       string `db:"role" json:"role"`
	TenantName string `db:"tenant_name" json:"tenant_name"`
	TenantSlug string `db:"tenant_slug" json:"tenant_slug"`
}
