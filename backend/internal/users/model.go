package users

import "time"

type User struct {
	ID        string    `json:"id" db:"id"`
	Auth0ID   string    `json:"-" db:"auth0_id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Email     string    `json:"email" db:"email"`
	FirstName string    `json:"firstname" db:"firstname"`
	LastName  string    `json:"lastname" db:"lastname"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
