package evaluation

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	flag "github.com/jalil32/toggle/internal/flags"
)

// Evaluator handles feature flag evaluation logic
type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate determines if a flag is enabled for the given context
// Returns false on any error (fail-safe behavior)
func (e *Evaluator) Evaluate(f *flag.Flag, ctx EvaluationContext) bool {
	// Step 1: If flag is globally disabled, return false immediately
	if !f.Enabled {
		return false
	}

	// Step 2: If no rules, return enabled state
	if len(f.Rules) == 0 {
		return f.Enabled
	}

	// Step 3: Evaluate all rules based on rule_logic (AND/OR)
	rulesPassed := e.evaluateRules(f, ctx)

	// Step 4: If rules failed, return false
	if !rulesPassed {
		return false
	}

	// Step 5: Apply rollout percentage using consistent hashing
	rolloutPercentage := e.getMaxRollout(f.Rules)
	userRolloutBucket := e.consistentHash(ctx.UserID, f.ID)

	return userRolloutBucket <= rolloutPercentage
}

// evaluateRules checks if rules pass based on AND/OR logic
func (e *Evaluator) evaluateRules(f *flag.Flag, ctx EvaluationContext) bool {
	if len(f.Rules) == 0 {
		return true
	}

	// Determine if AND or OR logic
	isAndLogic := f.RuleLogic == "AND"

	for _, rule := range f.Rules {
		matched := e.evaluateRule(rule, ctx)

		if isAndLogic && !matched {
			// AND: all must pass, early exit on first failure
			return false
		}
		if !isAndLogic && matched {
			// OR: any can pass, early exit on first success
			return true
		}
	}

	// AND: all passed (didn't early exit)
	// OR: none passed (didn't early exit)
	return isAndLogic
}

// evaluateRule checks if a single rule matches the context
func (e *Evaluator) evaluateRule(rule flag.Rule, ctx EvaluationContext) bool {
	// Get attribute value from context
	attrValue, exists := ctx.Attributes[rule.Attribute]
	if !exists {
		return false // Missing attribute = no match
	}

	switch rule.Operator {
	case "equals":
		return e.compareEquals(attrValue, rule.Value)
	case "not_equals":
		return !e.compareEquals(attrValue, rule.Value)
	case "in":
		return e.compareIn(attrValue, rule.Value)
	case "not_in":
		return !e.compareIn(attrValue, rule.Value)
	case "greater_than":
		return e.compareGreaterThan(attrValue, rule.Value)
	case "less_than":
		return e.compareLessThan(attrValue, rule.Value)
	default:
		// Unknown operator = fail-safe to false
		return false
	}
}

// compareEquals checks equality
func (e *Evaluator) compareEquals(attrValue, ruleValue interface{}) bool {
	return fmt.Sprintf("%v", attrValue) == fmt.Sprintf("%v", ruleValue)
}

// compareIn checks if attribute is in array
func (e *Evaluator) compareIn(attrValue, ruleValue interface{}) bool {
	// ruleValue should be an array
	arr, ok := ruleValue.([]interface{})
	if !ok {
		return false
	}

	attrStr := fmt.Sprintf("%v", attrValue)
	for _, v := range arr {
		if fmt.Sprintf("%v", v) == attrStr {
			return true
		}
	}
	return false
}

// compareGreaterThan for numeric comparisons
func (e *Evaluator) compareGreaterThan(attrValue, ruleValue interface{}) bool {
	attrNum, ok1 := e.toFloat64(attrValue)
	ruleNum, ok2 := e.toFloat64(ruleValue)
	if !ok1 || !ok2 {
		return false
	}
	return attrNum > ruleNum
}

// compareLessThan for numeric comparisons
func (e *Evaluator) compareLessThan(attrValue, ruleValue interface{}) bool {
	attrNum, ok1 := e.toFloat64(attrValue)
	ruleNum, ok2 := e.toFloat64(ruleValue)
	if !ok1 || !ok2 {
		return false
	}
	return attrNum < ruleNum
}

// toFloat64 converts interface{} to float64
func (e *Evaluator) toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// getMaxRollout finds the maximum rollout percentage from all rules
// (assumes all rules have the same rollout, uses first rule's value)
func (e *Evaluator) getMaxRollout(rules []flag.Rule) int {
	if len(rules) == 0 {
		return 100 // No rules = 100% rollout
	}
	return rules[0].Rollout
}

// consistentHash generates a deterministic 0-100 value from userID + flagID
// Same user + flag always returns same value
func (e *Evaluator) consistentHash(userID, flagID string) int {
	// Create deterministic hash input
	input := userID + ":" + flagID

	// SHA256 hash
	hash := sha256.Sum256([]byte(input))

	// Convert first 8 bytes to uint64
	hashInt := binary.BigEndian.Uint64(hash[:8])

	// Map to 0-100 range
	return int(hashInt % 101)
}
