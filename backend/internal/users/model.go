package users

import "time"

type User struct {
	ID                 string    `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Email              string    `json:"email" db:"email"`
	EmailVerified      bool      `json:"email_verified" db:"email_verified"`
	Image              *string   `json:"image,omitempty" db:"image"`
	LastActiveTenantID *string   `json:"last_active_tenant_id,omitempty" db:"last_active_tenant_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
