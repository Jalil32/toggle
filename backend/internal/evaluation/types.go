package evaluation

// define user context for feature flag evaluation
type EvalContext struct {
	UserID     string         `json:"user_id"`
	Attributes map[string]any `json:"attributes"`
}
