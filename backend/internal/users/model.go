package users

import "time"

type User struct {
	ID                 string    `json:"id" db:"id"`
	Auth0ID            string    `json:"-" db:"auth0_id"`
	LastActiveTenantID *string   `json:"last_active_tenant_id,omitempty" db:"last_active_tenant_id"`
	Email              string    `json:"email" db:"email"`
	FirstName          string    `json:"firstname" db:"firstname"`
	LastName           string    `json:"lastname" db:"lastname"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
