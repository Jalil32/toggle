package projects

import "time"

type Project struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	Name         string    `json:"name" db:"name"`
	ClientAPIKey string    `json:"client_api_key" db:"client_api_key"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type CreateRequest struct {
	Name string `json:"name" binding:"required"`
}
