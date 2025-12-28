package flag

import "time"

type Flag struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	ProjectID   *string   `json:"project_id,omitempty" db:"project_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	Rules       []Rule    `json:"rules" db:"rules"`
	RuleLogic   string    `json:"rule_logic" db:"rule_logic"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Rule struct {
	ID        string      `json:"id"`
	Attribute string      `json:"attribute"` // e.g., "country", "email"
	Operator  string      `json:"operator"`  // e.g., "equals", "contains", "in"
	Value     interface{} `json:"value"`     // e.g., "AU" or ["AU", "US"]
	Rollout   int         `json:"rollout"`   // 0-100 percentage
}
