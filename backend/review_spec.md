To ensure your AI agent reviews your multi-tenant refactor with a "security-first" mindset, you need a prompt that forces it to act like a Senior Security Engineer and a Go Architect.

Copy and paste this prompt to your agent. It is designed to find the specific "silent killers" of multi-tenant systems.

---

### The Agent Prompt

**Role:** Act as a Senior Go Backend Engineer and Security Auditor.
**Context:** We are refactoring our platform from a 1:1 user-org model to a Workspace-based Multi-tenancy model (using **Tenants**).

**Objective:** Conduct a comprehensive review of the current codebase against the "Tenant-First" specification and identify any violations of best practices or security vulnerabilities.

**Key Areas of Focus:**

1. **Cross-Tenant Data Leakage (High Priority):**
* Review all SQL queries in the `repository` layer. Every `SELECT`, `UPDATE`, and `DELETE` must include a `tenant_id` filter.
* Flag any queries that rely on `user_id` alone to fetch data that belongs to a tenant.
* Ensure that `tenant_id` is sourced strictly from the `context.Context` and never passed as a raw, unvalidated string from the handler.


2. **Middleware & Context Safety:**
* Verify the `TenantMiddleware` implementation. Does it correctly validate the `X-Tenant-ID` header against the `tenant_members` table?
* Is there a "Fail-Closed" policy? (i.e., if the header is missing or the membership check fails, does the request stop immediately with a 401/403?)
* Ensure context keys are defined as private types in `internal/pkg/contexts` to prevent collisions.


3. **Database Integrity:**
* Check that all tenant-specific tables have `FOREIGN KEY` constraints to the `tenants` table with `ON DELETE CASCADE`.
* Identify missing composite indexes—specifically `(tenant_id, id)` or `(user_id, tenant_id)`.


4. **Cache Invalidation (Race Conditions):**
* Review the Redis caching logic for memberships.
* Check for "stale cache" vulnerabilities: Are we deleting the `user:mems:{id}` key in every single place where a member is added, removed, or their role is changed?


5. **Go Best Practices:**
* Are database operations wrapped in transactions where multiple tables are hit (e.g., Signup, Invite Acceptance)?
* Are we using the standard `internal/pkg` structure for shared utilities like slugs and context helpers?



**Output Format:**
For every issue found, provide:

* **File Path & Line Number**
* **Severity:** (Critical, High, Medium, Low)
* **Description:** Why this is a vulnerability or bad practice.
* **Recommended Fix:** Specific code changes to resolve the issue.

