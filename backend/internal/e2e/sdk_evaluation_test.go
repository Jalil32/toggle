package e2e_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jalil32/toggle/internal/evaluation"
	flagspkg "github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/middleware"
	"github.com/jalil32/toggle/internal/projects"
	"github.com/jalil32/toggle/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateAPIKey generates a 64-character hex API key
func generateAPIKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read on most systems cannot fail
	return hex.EncodeToString(b)
}

// TestE2E_SDKEvaluationFlow simulates a complete SDK evaluation journey:
// 1. Create tenant, project, and multiple flags with different rules
// 2. Use the project's API key to authenticate
// 3. Test bulk evaluation endpoint (POST /sdk/evaluate)
// 4. Test single flag evaluation endpoint (POST /sdk/flags/:id/evaluate)
// 5. Verify results match expected evaluations
func TestE2E_SDKEvaluationFlow(t *testing.T) {
	db := testutil.GetTestDB()

	// Initialize services and middleware
	projectRepo := projects.NewRepository(db)
	flagRepo := flagspkg.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	evalService := evaluation.NewService(flagRepo, logger)
	evalHandler := evaluation.NewHandler(evalService)

	// Setup Gin router with SDK routes
	gin.SetMode(gin.TestMode)
	router := gin.New()

	sdk := router.Group("/sdk")
	sdk.Use(middleware.APIKey(projectRepo, logger))
	evalHandler.RegisterRoutes(sdk)

	// Start a transaction for data setup, commit it so middleware can see the data
	tx, err := db.Beginx()
	require.NoError(t, err)
	defer func() {
		// Cleanup: delete all test data at the end
		_ = tx.Rollback() // Ignore error in defer cleanup
		_, _ = db.Exec("DELETE FROM flags WHERE project_id IN (SELECT id FROM projects WHERE name = 'SDK Test Project')")
		_, _ = db.Exec("DELETE FROM projects WHERE name = 'SDK Test Project'")
		_, _ = db.Exec("DELETE FROM tenants WHERE slug = 'sdk-test-tenant'")
	}()

	// === STEP 1: Setup test data ===
	tenant := testutil.CreateTenant(t, tx, "SDK Test Tenant", "sdk-test-tenant")
	apiKey := generateAPIKey()
	project := testutil.CreateProject(t, tx, tenant.ID, "SDK Test Project", apiKey)

	// Create multiple flags with different configurations

	// Flag 1: Simple enabled flag with no rules (should always return true)
	flag1 := testutil.CreateFlag(t, tx, project.ID, "simple-flag", "Simple enabled flag", true)

	// Flag 2: Disabled flag (should always return false)
	flag2 := testutil.CreateFlag(t, tx, project.ID, "disabled-flag", "Disabled flag", false)

	// Flag 3: Country-based flag with AND logic
	rules3, _ := json.Marshal([]flagspkg.Rule{
		{
			ID:        "rule1",
			Attribute: "country",
			Operator:  "equals",
			Value:     "US",
			Rollout:   100,
		},
	})
	flag3 := testutil.CreateFlagWithRules(t, tx, project.ID, "country-flag", "Enabled only for US users", true, string(rules3), "AND")

	// Flag 4: Premium users OR specific countries (OR logic)
	rules4, _ := json.Marshal([]flagspkg.Rule{
		{
			ID:        "rule1",
			Attribute: "premium",
			Operator:  "equals",
			Value:     true,
			Rollout:   100,
		},
		{
			ID:        "rule2",
			Attribute: "country",
			Operator:  "in",
			Value:     []interface{}{"AU", "GB"},
			Rollout:   100,
		},
	})
	flag4 := testutil.CreateFlagWithRules(t, tx, project.ID, "premium-or-country-flag", "Enabled for premium users OR users in AU/GB", true, string(rules4), "OR")

	// Flag 5: Age-based flag with rollout percentage
	rules5, _ := json.Marshal([]flagspkg.Rule{
		{
			ID:        "rule1",
			Attribute: "age",
			Operator:  "greater_than",
			Value:     18,
			Rollout:   50,
		},
	})
	flag5 := testutil.CreateFlagWithRules(t, tx, project.ID, "age-based-flag", "Enabled for users over 18 with 50% rollout", true, string(rules5), "AND")

	// Commit the transaction so the middleware can see the data
	err = tx.Commit()
	require.NoError(t, err)

	// === STEP 2: Test Bulk Evaluation (POST /sdk/evaluate) ===
	t.Run("BulkEvaluation_MatchingContext", func(t *testing.T) {
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID: "test-user-123",
				Attributes: map[string]interface{}{
					"country": "US",
					"premium": false,
					"age":     25,
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp evaluation.EvaluationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Verify all flags are evaluated
		assert.Len(t, resp.Flags, 5, "Should evaluate all 5 flags")

		// Verify individual flag results
		assert.True(t, resp.Flags[flag1.ID], "Simple enabled flag should be true")
		assert.False(t, resp.Flags[flag2.ID], "Disabled flag should be false")
		assert.True(t, resp.Flags[flag3.ID], "Country flag should be true for US users")
		assert.False(t, resp.Flags[flag4.ID], "OR flag should be false (not premium AND not AU/GB)")
		// flag5 depends on rollout hash, so we just verify it's evaluated
		assert.Contains(t, resp.Flags, flag5.ID, "Age-based flag should be present")

		t.Logf("Bulk evaluation results: %+v", resp.Flags)
	})

	t.Run("BulkEvaluation_DifferentContext", func(t *testing.T) {
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID: "premium-user-456",
				Attributes: map[string]interface{}{
					"country": "AU",
					"premium": true,
					"age":     30,
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp evaluation.EvaluationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Verify flag results for different context
		assert.True(t, resp.Flags[flag1.ID], "Simple enabled flag should be true")
		assert.False(t, resp.Flags[flag2.ID], "Disabled flag should be false")
		assert.False(t, resp.Flags[flag3.ID], "Country flag should be false for AU users")
		assert.True(t, resp.Flags[flag4.ID], "OR flag should be true (premium user)")
	})

	// === STEP 3: Test Single Flag Evaluation (POST /sdk/flags/:id/evaluate) ===
	t.Run("SingleFlagEvaluation_Success", func(t *testing.T) {
		reqBody := evaluation.SingleEvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID: "single-user-789",
				Attributes: map[string]interface{}{
					"country": "US",
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/flags/"+flag3.ID+"/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp evaluation.SingleEvaluationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, flag3.ID, resp.FlagID)
		assert.True(t, resp.Enabled, "Country flag should be enabled for US users")
	})

	t.Run("SingleFlagEvaluation_NotFound", func(t *testing.T) {
		reqBody := evaluation.SingleEvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID:     "test-user",
				Attributes: map[string]interface{}{},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/flags/nonexistent-flag-id/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// === STEP 4: Test Invalid API Key ===
	t.Run("InvalidAPIKey_Returns401", func(t *testing.T) {
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID:     "test-user",
				Attributes: map[string]interface{}{},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid-api-key-12345")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("MissingAPIKey_Returns401", func(t *testing.T) {
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID:     "test-user",
				Attributes: map[string]interface{}{},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// === STEP 5: Test Rollout Consistency ===
	t.Run("RolloutConsistency_SameUserSameResult", func(t *testing.T) {
		userID := "consistent-user-999"
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID: userID,
				Attributes: map[string]interface{}{
					"age": 25,
				},
			},
		}

		// Call evaluation multiple times
		var results []bool
		for i := 0; i < 5; i++ {
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var resp evaluation.EvaluationResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			results = append(results, resp.Flags[flag5.ID])
		}

		// All results should be identical (consistent hashing)
		firstResult := results[0]
		for _, result := range results {
			assert.Equal(t, firstResult, result, "Same user should get consistent evaluation results")
		}
	})
}
