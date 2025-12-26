# Feature Flag Evaluation System - Implementation Plan

## Overview
Implement a feature flag evaluation system with SDK authentication using `client_api_key`, consistent hashing for rollout percentages, and configurable AND/OR rule logic.

## Design Decisions
- **Consistent Hashing**: SHA256(userID:flagID) → deterministic 0-100 value
- **Flag Identifier**: Use `flag.id` (UUID) in SDK responses
- **Failure Mode**: Default to `enabled: false` (fail-safe)
- **Rule Logic**: Configurable per flag via new `rule_logic` field ("AND"/"OR")

## Implementation Phases

### Phase 1: Database Schema (Migration + Flag Model)

1. **Create Migration**: `migrations/20251225HHMMSS_add_rule_logic_to_flags.sql`
   - Add `rule_logic VARCHAR(10) NOT NULL DEFAULT 'AND'`
   - Add CHECK constraint for only 'AND'/'OR' values
   - Create index on rule_logic
   - Include goose Up/Down blocks

2. **Update Flag Model**: `internal/flags/model.go`
   - Add `RuleLogic string` field with db and json tags

3. **Update Flag Repository**: `internal/flags/repository.go`
   - Update ALL queries to include `rule_logic` field:
     - Create (line ~44)
     - GetByID (line ~63)
     - List (line ~89)
     - ListByProject (line ~133)
     - Update (line ~184)

4. **Update Flag Service/Handler**: `internal/flags/service.go`, `internal/flags/handler.go`
   - Add `RuleLogic` to CreateRequest and UpdateRequest
   - Set default "AND" if not provided

### Phase 2: API Key Authentication

1. **Add GetByAPIKey to Projects Repository**: `internal/projects/repository.go`
   - Add method to Repository interface
   - Query: `SELECT * FROM projects WHERE client_api_key = $1`
   - No tenant_id in WHERE (API key is the auth mechanism)

2. **Create API Key Middleware**: `internal/middleware/apikey.go`
   - Extract Bearer token from Authorization header
   - Call `projectRepo.GetByAPIKey(apiKey)`
   - Inject project_id and tenant_id into context via `appContext.WithSDKAuth`
   - Return 401 for invalid/missing API key

3. **Extend Context Helpers**: `internal/pkg/context/tenant.go`
   - Add `projectIDKey` context key
   - Add `WithSDKAuth(ctx, projectID, tenantID)` function
   - Add `ProjectID(ctx)` and `MustProjectID(ctx)` helpers

### Phase 3: Evaluation Engine

1. **Create Evaluation Types**: `internal/evaluation/types.go`
   ```go
   type EvaluationContext struct {
       UserID     string
       Attributes map[string]interface{}
   }

   type EvaluationRequest struct {
       Context EvaluationContext
   }

   type EvaluationResponse struct {
       Flags map[string]bool  // map[flag_id]enabled
   }
   ```

2. **Create Evaluator**: `internal/evaluation/evaluator.go`
   - **Evaluate(flag, context)** - Main evaluation logic
   - **consistentHash(userID, flagID)** - SHA256 hash to 0-100 range
   - **evaluateRules(flag, context)** - AND/OR logic dispatcher
   - **evaluateRule(rule, context)** - Single rule evaluation
   - **Operators**: equals, not_equals, in, not_in, greater_than, less_than
   - **Fail-safe**: Return false on any error or unknown operator

3. **Create Evaluation Service**: `internal/evaluation/service.go`
   - **EvaluateAll(projectID, evalContext)** - Bulk evaluation
     - Fetch all flags via `flagRepo.ListByProject`
     - Evaluate each flag
     - Return map[flag_id]bool
   - **EvaluateSingle(flagID, tenantID, evalContext)** - Single flag
     - Fetch flag via `flagRepo.GetByID`
     - Evaluate and return result

4. **Create Evaluation Handler**: `internal/evaluation/handler.go`
   - `POST /evaluate` - Bulk evaluation endpoint
   - `POST /flags/:id/evaluate` - Single flag evaluation
   - Extract project_id/tenant_id from context (set by middleware)

### Phase 4: Route Registration

**Update**: `internal/routes/routes.go`

Add SDK route group BEFORE protected routes:
```go
// SDK routes (API key authentication, no Auth0)
sdk := api.Group("/sdk")
sdk.Use(middleware.APIKey(projectRepo, logger))
{
    evaluationHandler.RegisterRoutes(sdk)
}
```

Endpoints:
- `POST /api/v1/sdk/evaluate` - Bulk evaluation
- `POST /api/v1/sdk/flags/:id/evaluate` - Single evaluation

### Phase 5: Testing

1. **Evaluator Tests**: `internal/evaluation/evaluator_test.go`
   - Consistent hashing is deterministic
   - Disabled flags always return false
   - No rules = enabled
   - Operator tests (equals, in, greater_than, etc.)
   - AND vs OR logic
   - Rollout percentage distribution

2. **E2E SDK Test**: `internal/e2e/sdk_evaluation_test.go`
   - Complete flow: GetByAPIKey → EvaluateAll → Verify results
   - Test multiple flags with different rules
   - Test with matching and non-matching contexts

3. **Security Tests**: `internal/security/apikey_security_test.go`
   - Invalid API key rejection
   - SQL injection safety
   - Tenant isolation via API key

4. **Update Fixtures**: `internal/testutil/fixtures.go`
   - Add `RuleLogic string` to testutil.Flag struct
   - Update `CreateFlag` to include rule_logic in INSERT

## Critical Files

### New Files (9)
1. `migrations/20251225HHMMSS_add_rule_logic_to_flags.sql`
2. `internal/middleware/apikey.go`
3. `internal/evaluation/types.go`
4. `internal/evaluation/evaluator.go`
5. `internal/evaluation/service.go`
6. `internal/evaluation/handler.go`
7. `internal/evaluation/evaluator_test.go`
8. `internal/e2e/sdk_evaluation_test.go`
9. `internal/security/apikey_security_test.go`

### Modified Files (8)
1. `internal/flags/model.go` - Add RuleLogic field
2. `internal/flags/repository.go` - Update all queries
3. `internal/flags/service.go` - Add RuleLogic to requests
4. `internal/flags/handler.go` - Handle RuleLogic
5. `internal/projects/repository.go` - Add GetByAPIKey
6. `internal/pkg/context/tenant.go` - Add SDK context helpers
7. `internal/routes/routes.go` - Register SDK routes
8. `internal/testutil/fixtures.go` - Update CreateFlag

## Key Implementation Details

### Consistent Hashing Algorithm
```go
func consistentHash(userID, flagID string) int {
    input := userID + ":" + flagID
    hash := sha256.Sum256([]byte(input))
    hashInt := binary.BigEndian.Uint64(hash[:8])
    return int(hashInt % 101)  // 0-100 range
}
```

### AND vs OR Rule Logic
```go
if flag.RuleLogic == "OR" {
    // ANY rule can pass
    for _, rule := range flag.Rules {
        if evaluateRule(rule, ctx) && checkRollout(...) {
            return true
        }
    }
    return false
} else {  // "AND"
    // ALL rules must pass
    for _, rule := range flag.Rules {
        if !evaluateRule(rule, ctx) || !checkRollout(...) {
            return false
        }
    }
    return true
}
```

### API Key Authentication Flow
```
1. SDK sends: Authorization: Bearer {client_api_key}
2. Middleware extracts API key
3. Lookup project by API key → get project_id + tenant_id
4. Inject into context via WithSDKAuth(project_id, tenant_id)
5. Handler uses MustProjectID(ctx) to get project
6. Service fetches flags with tenant scoping
```

## Security Considerations
- ✅ All flag queries join with projects for tenant scoping
- ✅ API key lookup is SQL injection safe (parameterized)
- ✅ Invalid API keys return 401 (not 404 to prevent enumeration)
- ✅ Evaluation failures default to false (fail-safe)
- ✅ No Auth0 required for SDK endpoints (separate middleware chain)

## Testing Strategy
- Unit tests for evaluator logic (operators, hashing, AND/OR)
- Sociable tests for service + repository integration
- E2E test for complete SDK flow
- Security tests for API key validation and tenant isolation
- Use `testutil.WithTestDB` for all database tests

## Execution Order
1. Phase 1 (Database) - Foundation for everything
2. Phase 2 (API Key Auth) - Required for SDK endpoints
3. Phase 3 (Evaluation Engine) - Core business logic
4. Phase 4 (Routes) - Wire everything together
5. Phase 5 (Testing) - Validate implementation

Run migration first, then run tests after each phase to ensure progress.
