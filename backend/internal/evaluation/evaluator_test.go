package evaluation

import (
	"fmt"
	"testing"

	flag "github.com/jalil32/toggle/internal/flags"
	"github.com/stretchr/testify/assert"
)

func TestEvaluator_ConsistentHash_IsDeterministic(t *testing.T) {
	e := NewEvaluator()

	userID := "user123"
	flagID := "flag456"

	// Call multiple times, should always return same value
	hash1 := e.consistentHash(userID, flagID)
	hash2 := e.consistentHash(userID, flagID)
	hash3 := e.consistentHash(userID, flagID)

	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
	assert.GreaterOrEqual(t, hash1, 0)
	assert.LessOrEqual(t, hash1, 100)
}

func TestEvaluator_ConsistentHash_DifferentUsers(t *testing.T) {
	e := NewEvaluator()

	flagID := "flag123"

	// Test with multiple users to ensure hashing distributes values
	hashes := make(map[int]bool)
	for i := 0; i < 10; i++ {
		hash := e.consistentHash(fmt.Sprintf("user%d", i), flagID)
		hashes[hash] = true
	}

	// With 10 different users, we should get at least 2 different hash values
	// (statistically extremely likely with good hash function)
	assert.GreaterOrEqual(t, len(hashes), 2, "Different users should produce different hashes")
}

func TestEvaluator_DisabledFlag_ReturnsFalse(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:      "flag1",
		Enabled: false,
		Rules:   []flag.Rule{},
	}

	ctx := EvaluationContext{
		UserID:     "user1",
		Attributes: map[string]interface{}{},
	}

	result := e.Evaluate(f, ctx)
	assert.False(t, result, "Disabled flag should always return false")
}

func TestEvaluator_NoRules_ReturnsEnabled(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:      "flag1",
		Enabled: true,
		Rules:   []flag.Rule{},
	}

	ctx := EvaluationContext{
		UserID:     "user1",
		Attributes: map[string]interface{}{},
	}

	result := e.Evaluate(f, ctx)
	assert.True(t, result, "Enabled flag with no rules should return true")
}

func TestEvaluator_Operator_Equals(t *testing.T) {
	e := NewEvaluator()

	tests := []struct {
		name       string
		attrValue  interface{}
		ruleValue  interface{}
		shouldPass bool
	}{
		{"string match", "US", "US", true},
		{"string no match", "US", "AU", false},
		{"int match", 5, 5, true},
		{"int no match", 5, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &flag.Flag{
				ID:        "flag1",
				Enabled:   true,
				RuleLogic: "AND",
				Rules: []flag.Rule{
					{
						Attribute: "country",
						Operator:  "equals",
						Value:     tt.ruleValue,
						Rollout:   100,
					},
				},
			}

			ctx := EvaluationContext{
				UserID: "user1",
				Attributes: map[string]interface{}{
					"country": tt.attrValue,
				},
			}

			result := e.Evaluate(f, ctx)
			assert.Equal(t, tt.shouldPass, result)
		})
	}
}

func TestEvaluator_Operator_NotEquals(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "not_equals",
				Value:     "US",
				Rollout:   100,
			},
		},
	}

	// Should pass for non-US
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "AU",
		},
	}
	assert.True(t, e.Evaluate(f, ctx))

	// Should fail for US
	ctx.Attributes["country"] = "US"
	assert.False(t, e.Evaluate(f, ctx))
}

func TestEvaluator_Operator_In(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "in",
				Value:     []interface{}{"US", "AU", "GB"},
				Rollout:   100,
			},
		},
	}

	// Should pass for countries in list
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "AU",
		},
	}
	assert.True(t, e.Evaluate(f, ctx))

	// Should fail for countries not in list
	ctx.Attributes["country"] = "FR"
	assert.False(t, e.Evaluate(f, ctx))
}

func TestEvaluator_Operator_NotIn(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "not_in",
				Value:     []interface{}{"US", "AU"},
				Rollout:   100,
			},
		},
	}

	// Should pass for countries not in list
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "FR",
		},
	}
	assert.True(t, e.Evaluate(f, ctx))

	// Should fail for countries in list
	ctx.Attributes["country"] = "US"
	assert.False(t, e.Evaluate(f, ctx))
}

func TestEvaluator_Operator_GreaterThan(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "age",
				Operator:  "greater_than",
				Value:     18,
				Rollout:   100,
			},
		},
	}

	// Should pass for values > 18
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"age": 25,
		},
	}
	assert.True(t, e.Evaluate(f, ctx))

	// Should fail for values <= 18
	ctx.Attributes["age"] = 18
	assert.False(t, e.Evaluate(f, ctx))

	ctx.Attributes["age"] = 10
	assert.False(t, e.Evaluate(f, ctx))
}

func TestEvaluator_Operator_LessThan(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "age",
				Operator:  "less_than",
				Value:     65,
				Rollout:   100,
			},
		},
	}

	// Should pass for values < 65
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"age": 30,
		},
	}
	assert.True(t, e.Evaluate(f, ctx))

	// Should fail for values >= 65
	ctx.Attributes["age"] = 65
	assert.False(t, e.Evaluate(f, ctx))

	ctx.Attributes["age"] = 70
	assert.False(t, e.Evaluate(f, ctx))
}

func TestEvaluator_RuleLogic_AND(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "equals",
				Value:     "US",
				Rollout:   100,
			},
			{
				Attribute: "premium",
				Operator:  "equals",
				Value:     true,
				Rollout:   100,
			},
		},
	}

	// Both rules pass
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "US",
			"premium": true,
		},
	}
	assert.True(t, e.Evaluate(f, ctx), "AND logic should pass when all rules pass")

	// First rule passes, second fails
	ctx.Attributes["premium"] = false
	assert.False(t, e.Evaluate(f, ctx), "AND logic should fail when any rule fails")

	// First rule fails, second passes
	ctx.Attributes["country"] = "AU"
	ctx.Attributes["premium"] = true
	assert.False(t, e.Evaluate(f, ctx), "AND logic should fail when any rule fails")

	// Both rules fail
	ctx.Attributes["country"] = "AU"
	ctx.Attributes["premium"] = false
	assert.False(t, e.Evaluate(f, ctx), "AND logic should fail when all rules fail")
}

func TestEvaluator_RuleLogic_OR(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "OR",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "equals",
				Value:     "US",
				Rollout:   100,
			},
			{
				Attribute: "premium",
				Operator:  "equals",
				Value:     true,
				Rollout:   100,
			},
		},
	}

	// Both rules pass
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "US",
			"premium": true,
		},
	}
	assert.True(t, e.Evaluate(f, ctx), "OR logic should pass when all rules pass")

	// First rule passes, second fails
	ctx.Attributes["premium"] = false
	assert.True(t, e.Evaluate(f, ctx), "OR logic should pass when any rule passes")

	// First rule fails, second passes
	ctx.Attributes["country"] = "AU"
	ctx.Attributes["premium"] = true
	assert.True(t, e.Evaluate(f, ctx), "OR logic should pass when any rule passes")

	// Both rules fail
	ctx.Attributes["country"] = "AU"
	ctx.Attributes["premium"] = false
	assert.False(t, e.Evaluate(f, ctx), "OR logic should fail when all rules fail")
}

func TestEvaluator_MissingAttribute_ReturnsFalse(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "equals",
				Value:     "US",
				Rollout:   100,
			},
		},
	}

	// Missing the "country" attribute
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"premium": true,
		},
	}

	result := e.Evaluate(f, ctx)
	assert.False(t, result, "Missing attribute should fail evaluation")
}

func TestEvaluator_UnknownOperator_ReturnsFalse(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "unknown_operator",
				Value:     "US",
				Rollout:   100,
			},
		},
	}

	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "US",
		},
	}

	result := e.Evaluate(f, ctx)
	assert.False(t, result, "Unknown operator should fail-safe to false")
}

func TestEvaluator_Rollout_Distribution(t *testing.T) {
	e := NewEvaluator()

	// Create flag with 50% rollout
	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "equals",
				Value:     "US",
				Rollout:   50,
			},
		},
	}

	// Test with 100 different users
	enabled := 0
	disabled := 0

	for i := 0; i < 100; i++ {
		ctx := EvaluationContext{
			UserID: fmt.Sprintf("user%d", i),
			Attributes: map[string]interface{}{
				"country": "US",
			},
		}

		if e.Evaluate(f, ctx) {
			enabled++
		} else {
			disabled++
		}
	}

	// With 50% rollout, we expect roughly 50 enabled and 50 disabled
	// Allow some variance (30-70 range is reasonable)
	assert.GreaterOrEqual(t, enabled, 30, "Should have at least 30% enabled")
	assert.LessOrEqual(t, enabled, 70, "Should have at most 70% enabled")
}

func TestEvaluator_Rollout_0Percent(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "equals",
				Value:     "US",
				Rollout:   0,
			},
		},
	}

	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"country": "US",
		},
	}

	// 0% rollout should always return false
	result := e.Evaluate(f, ctx)
	assert.False(t, result, "0% rollout should always return false")
}

func TestEvaluator_Rollout_100Percent(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "country",
				Operator:  "equals",
				Value:     "US",
				Rollout:   100,
			},
		},
	}

	// Test with multiple users, all should pass
	for i := 0; i < 10; i++ {
		ctx := EvaluationContext{
			UserID: fmt.Sprintf("user%d", i),
			Attributes: map[string]interface{}{
				"country": "US",
			},
		}

		result := e.Evaluate(f, ctx)
		assert.True(t, result, "100%% rollout should always return true for matching rules")
	}
}

func TestEvaluator_NumericComparison_WithFloat64(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "score",
				Operator:  "greater_than",
				Value:     float64(75.5),
				Rollout:   100,
			},
		},
	}

	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"score": float64(80.0),
		},
	}

	assert.True(t, e.Evaluate(f, ctx))

	ctx.Attributes["score"] = float64(70.0)
	assert.False(t, e.Evaluate(f, ctx))
}

func TestEvaluator_NumericComparison_InvalidType(t *testing.T) {
	e := NewEvaluator()

	f := &flag.Flag{
		ID:        "flag1",
		Enabled:   true,
		RuleLogic: "AND",
		Rules: []flag.Rule{
			{
				Attribute: "age",
				Operator:  "greater_than",
				Value:     18,
				Rollout:   100,
			},
		},
	}

	// Provide string instead of number
	ctx := EvaluationContext{
		UserID: "user1",
		Attributes: map[string]interface{}{
			"age": "not_a_number",
		},
	}

	result := e.Evaluate(f, ctx)
	assert.False(t, result, "Invalid numeric type should fail comparison")
}
