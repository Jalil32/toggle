This testing specification ensures your multi-tenant refactor is bulletproof. Since you are moving to a **Tenant-based** model, the goal is to prove that the "Tenant Boundary" cannot be crossed, even if IDs are known.

---

# Multi-Tenancy Testing Specification

## 1. Unit Testing: The "Security Gate"

Focus on the middleware and slug utilities. These are the entry points where `tenant_id` is resolved.

* **Middleware Logic Tests:**
* **Case A:** Valid Auth0 JWT + Valid `X-Tenant-ID` header  **Success (200)**. Check that `ctx` contains correct `tenant_id`.
* **Case B:** Valid Auth0 JWT + Missing `X-Tenant-ID`  **Failure (400 Bad Request)**.
* **Case C:** Valid Auth0 JWT + `X-Tenant-ID` for a tenant the user *does not* belong to  **Failure (403 Forbidden)**.
* **Case D:** Valid Auth0 JWT + Randomly generated UUID for `X-Tenant-ID`  **Failure (403 Forbidden)**.


* **Slug Generation Tests:**
* Input: "Acme Corp!"  Output: `acme-corp`.
* Input: "   Space   "  Output: `space`.
* Input: "2025 Team 🎉"  Output: `2025-team`.



---

## 2. Integration Testing: "The Leak Proof"

These tests require a live database (ideally in a Docker container using `testcontainers-go`).

### Test Setup (The "Sandwich" Strategy)

1. **Seed Tenant A:** Create `Tenant_A`, `User_A` (Owner), and `Project_A1`.
2. **Seed Tenant B:** Create `Tenant_B`, `User_B` (Owner), and `Project_B1`.

### The "Isolation" Test Suite

| Test Case | Actor | Action | Expected Result |
| --- | --- | --- | --- |
| **Direct Access** | `User_A` | `GET /projects/Project_A1` | `200 OK` |
| **ID Guessing** | `User_A` | `GET /projects/Project_B1` (using `X-Tenant-ID: Tenant_A`) | `404 Not Found` or `403 Forbidden` |
| **Header Swapping** | `User_A` | `GET /projects/Project_B1` (using `X-Tenant-ID: Tenant_B`) | `403 Forbidden` (Membership check fails) |
| **Global Listing** | `User_A` | `GET /projects` | Returns `[Project_A1]`, **never** `Project_B1`. |

---

## 3. Cache Testing: "Data Freshness"

Since we are using Redis to save database trips, we must ensure the cache doesn't serve stale membership data.

* **Positive Cache Test:**
1. Request `GET /projects` for `User_A`.
2. Verify DB query was logged.
3. Repeat request.
4. Verify DB query was **not** logged (Cache Hit).


* **Invalidation Test:**
1. `User_A` is in `Tenant_A`. Cache is warm.
2. An Admin removes `User_A` from `Tenant_A`.
3. `User_A` requests `GET /projects` again.
4. Verify response is `403 Forbidden` (Cache was successfully purged on removal).



---

## 4. End-to-End (E2E) Flow: "The Lifecycle"

This mimics a real user journey.

1. **Onboarding:** Call the signup endpoint. Verify a `User`, a `Tenant`, and a `TenantMember` are created.
2. **Invitation:** `User_A` invites `user_b@example.com` to `Tenant_A`. Verify an invitation token is generated.
3. **Acceptance:** `User_B` calls `/accept/:token`. Verify `User_B` can now access `Tenant_A` projects using the header.
4. **Multi-Tenant Switch:** Verify `User_B` (who now belongs to two orgs) gets both IDs back when calling `GET /api/v1/organizations`.

---

## 5. Repository Layer "Sanity Check"

To ensure developers include `tenant_id` in every SQL query, we use a **Database Scoping Test**.

```go
// Example Test Logic
func TestRepositoryAlwaysScopes(t *testing.T) {
    repo := NewProjectRepository(db)
    // Attempt to fetch a project without a tenant_id in the context
    _, err := repo.GetByID(context.Background(), someID) 
    
    // The repository should return an error if the context is missing tenant info
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "tenant_id missing from context")
}

```

---

## 6. Testing Guardrails (Pre-Deployment)

* **SQL Audit:** Run a regex search across the codebase for `SELECT` statements. Ensure every statement contains the string `tenant_id`.
* **Race Conditions:** Use the `-race` flag during tests to ensure your Context and Redis handling are thread-safe.

**Next Step:**
Would you like me to provide a **Go code template for the "Isolation" integration test** so your agent can begin implementing the safety checks?
