package security_test

import (
	"bytes"
	"context"
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
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateAPIKey generates a 64-character hex API key for testing
func generateAPIKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read on most systems cannot fail
	return hex.EncodeToString(b)
}

// TestAPIKey_InvalidKey_Returns401 tests that invalid API keys are rejected
func TestAPIKey_InvalidKey_Returns401(t *testing.T) {
	db := testutil.GetTestDB()
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

	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID:     "test-user",
				Attributes: map[string]interface{}{},
			},
		}

		// Test 1: Completely fake API key
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer this-is-a-fake-api-key-12345678901234567890123456789012")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Fake API key should return 401")

		// Test 2: Missing Authorization header entirely
		req = httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Missing API key should return 401")

		// Test 3: Malformed Authorization header (no Bearer prefix)
		req = httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "this-is-a-fake-api-key")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Malformed API key should return 401")

		// Test 4: Empty Bearer token
		req = httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer ")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Empty API key should return 401")
	})
}

// TestAPIKey_SQLInjection_Safe tests that API key lookup is safe from SQL injection
func TestAPIKey_SQLInjection_Safe(t *testing.T) {
	db := testutil.GetTestDB()
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

	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		reqBody := evaluation.EvaluationRequest{
			Context: evaluation.EvaluationContext{
				UserID:     "test-user",
				Attributes: map[string]interface{}{},
			},
		}
		body, _ := json.Marshal(reqBody)

		// SQL Injection attempts that should all be safely rejected
		sqlInjectionAttempts := []string{
			"' OR '1'='1",
			"'; DROP TABLE projects; --",
			"' UNION SELECT * FROM tenants --",
			"1' OR '1'='1' --",
			"admin'--",
			"' OR 1=1 UNION SELECT NULL, NULL, NULL --",
			"'; DELETE FROM flags WHERE '1'='1",
		}

		for _, injectionAttempt := range sqlInjectionAttempts {
			req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+injectionAttempt)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// All injection attempts should safely return 401 (not crash or succeed)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"SQL injection attempt should return 401: %s", injectionAttempt)
		}

		// Verify database is still intact after injection attempts
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM projects")
		require.NoError(t, err, "Database should still be queryable after injection attempts")
	})
}

// TestAPIKey_TenantIsolation_StrictSeparation tests that API keys provide strict tenant isolation
func TestAPIKey_TenantIsolation_StrictSeparation(t *testing.T) {
	db := testutil.GetTestDB()
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

	// Setup: Create two separate transactions and commit them so data is visible
	tx1, err := db.Beginx()
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("DELETE FROM flags WHERE project_id IN (SELECT id FROM projects WHERE name IN ('Tenant1 Project', 'Tenant2 Project'))")
		_, _ = db.Exec("DELETE FROM projects WHERE name IN ('Tenant1 Project', 'Tenant2 Project')")
		_, _ = db.Exec("DELETE FROM tenants WHERE slug IN ('tenant1-test', 'tenant2-test')")
	}()

	// Create Tenant 1 with project and flag
	tenant1 := testutil.CreateTenant(t, tx1, "Tenant 1", "tenant1-test")
	apiKey1 := generateAPIKey()
	project1 := testutil.CreateProject(t, tx1, tenant1.ID, "Tenant1 Project", apiKey1)
	flag1 := testutil.CreateFlag(t, tx1, tenant1.ID, &project1.ID, "tenant1-flag", "Tenant 1 Flag", true)

	// Create Tenant 2 with project and flag
	tenant2 := testutil.CreateTenant(t, tx1, "Tenant 2", "tenant2-test")
	apiKey2 := generateAPIKey()
	project2 := testutil.CreateProject(t, tx1, tenant2.ID, "Tenant2 Project", apiKey2)
	flag2 := testutil.CreateFlag(t, tx1, tenant2.ID, &project2.ID, "tenant2-flag", "Tenant 2 Flag", true)

	err = tx1.Commit()
	require.NoError(t, err)

	// Test 1: Tenant 1 API key should only see Tenant 1 flags
	reqBody := evaluation.EvaluationRequest{
		Context: evaluation.EvaluationContext{
			UserID:     "test-user",
			Attributes: map[string]interface{}{},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp1 evaluation.EvaluationResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp1)
	require.NoError(t, err)

	// Verify Tenant 1 can see their own flag
	assert.Contains(t, resp1.Flags, flag1.ID, "Tenant 1 should see their own flag")

	// Verify Tenant 1 CANNOT see Tenant 2's flag
	assert.NotContains(t, resp1.Flags, flag2.ID, "Tenant 1 should NOT see Tenant 2's flag")
	assert.Len(t, resp1.Flags, 1, "Tenant 1 should only see 1 flag (their own)")

	// Test 2: Tenant 2 API key should only see Tenant 2 flags
	req = httptest.NewRequest(http.MethodPost, "/sdk/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey2)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp2 evaluation.EvaluationResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp2)
	require.NoError(t, err)

	// Verify Tenant 2 can see their own flag
	assert.Contains(t, resp2.Flags, flag2.ID, "Tenant 2 should see their own flag")

	// Verify Tenant 2 CANNOT see Tenant 1's flag
	assert.NotContains(t, resp2.Flags, flag1.ID, "Tenant 2 should NOT see Tenant 1's flag")
	assert.Len(t, resp2.Flags, 1, "Tenant 2 should only see 1 flag (their own)")

	// Test 3: Attempt to access Tenant 1's specific flag with Tenant 2's API key
	singleReq := evaluation.SingleEvaluationRequest{
		Context: evaluation.EvaluationContext{
			UserID:     "test-user",
			Attributes: map[string]interface{}{},
		},
	}

	body, _ = json.Marshal(singleReq)
	req = httptest.NewRequest(http.MethodPost, "/sdk/flags/"+flag1.ID+"/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey2) // Using Tenant 2's key

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 (not found) to prevent ID enumeration
	assert.Equal(t, http.StatusNotFound, w.Code,
		"Cross-tenant flag access should return 404 (not 403 to prevent ID enumeration)")
}

// TestAPIKey_DirectRepositoryAccess_ParameterizedQueries tests that the
// GetByAPIKey repository method uses parameterized queries
func TestAPIKey_DirectRepositoryAccess_ParameterizedQueries(t *testing.T) {
	db := testutil.GetTestDB()
	projectRepo := projects.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Create a valid project
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")
		apiKey := generateAPIKey()
		project := testutil.CreateProject(t, tx, tenant.ID, "Test Project", apiKey)

		// Test 1: Valid API key lookup (should succeed)
		retrieved, err := projectRepo.GetByAPIKey(ctx, apiKey)
		require.NoError(t, err)
		assert.Equal(t, project.ID, retrieved.ID)
		assert.Equal(t, tenant.ID, retrieved.TenantID)

		// Test 2: SQL injection attempts should safely fail
		injectionAttempts := []string{
			"' OR '1'='1",
			apiKey + "' OR '1'='1",
			"'; DROP TABLE projects; --",
		}

		for _, injection := range injectionAttempts {
			retrieved, err := projectRepo.GetByAPIKey(ctx, injection)

			// Should return no rows (not crash or return unexpected data)
			assert.Error(t, err, "Injection attempt should fail: %s", injection)
			assert.Nil(t, retrieved, "Injection attempt should return nil: %s", injection)
		}
	})
}
