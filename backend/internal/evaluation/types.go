package evaluation

// EvaluationContext contains user attributes and context for evaluation
type EvaluationContext struct {
	UserID     string                 `json:"user_id" binding:"required"`
	Attributes map[string]interface{} `json:"attributes"`
}

// EvaluationRequest is the bulk evaluation request from SDK
type EvaluationRequest struct {
	Context EvaluationContext `json:"context" binding:"required"`
}

// EvaluationResponse returns all flag states for the user
type EvaluationResponse struct {
	Flags map[string]bool `json:"flags"` // map[flag_id]enabled
}

// SingleEvaluationRequest is for evaluating a single flag
type SingleEvaluationRequest struct {
	Context EvaluationContext `json:"context" binding:"required"`
}

// SingleEvaluationResponse returns the evaluation result for one flag
type SingleEvaluationResponse struct {
	Enabled bool   `json:"enabled"`
	FlagID  string `json:"flag_id"`
}
