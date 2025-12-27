-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Users table - Better Auth integration
-- Stores core user profile data with support for email-based auth
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    image TEXT,
    last_active_tenant_id UUID, -- FK added after tenants table
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tenants table - Workspaces/Organizations
-- Each tenant represents an isolated workspace with its own data
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add foreign key from users to tenants (after both tables exist)
ALTER TABLE users
    ADD CONSTRAINT fk_users_last_active_tenant
    FOREIGN KEY (last_active_tenant_id)
    REFERENCES tenants(id) ON DELETE SET NULL;

-- ============================================================================
-- MULTI-TENANCY TABLES
-- ============================================================================

-- Tenant members - Many-to-many relationship between users and tenants
CREATE TABLE tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, tenant_id)
);

-- Tenant invitations - Invite users to join tenants
CREATE TABLE tenant_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- ============================================================================
-- FEATURE FLAG TABLES
-- ============================================================================

-- Projects - Container for feature flags (scoped to tenant)
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    client_api_key VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Flags - Feature flags with rules and rollout configuration
CREATE TABLE flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT false,
    rules JSONB NOT NULL DEFAULT '[]',
    rule_logic VARCHAR(10) NOT NULL DEFAULT 'AND',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rule_logic_check CHECK (rule_logic IN ('AND', 'OR'))
);

-- ============================================================================
-- BETTER AUTH TABLES
-- ============================================================================

-- Sessions - User session management
CREATE TABLE session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Accounts - OAuth provider accounts and credentials
CREATE TABLE account (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    access_token_expires_at TIMESTAMPTZ,
    refresh_token_expires_at TIMESTAMPTZ,
    scope TEXT,
    id_token TEXT,
    password TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, provider_id)
);

-- Verification - Magic links, email verification, password reset tokens
CREATE TABLE verification (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier TEXT NOT NULL,
    value TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- JWKS - JSON Web Key Sets for JWT signing
CREATE TABLE jwks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- User indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_last_active_tenant ON users(last_active_tenant_id);

-- Tenant indexes
CREATE INDEX idx_tenants_slug ON tenants(slug);

-- Tenant membership indexes
CREATE INDEX idx_tenant_members_user ON tenant_members(user_id);
CREATE INDEX idx_tenant_members_tenant ON tenant_members(tenant_id);

-- Tenant invitation indexes
CREATE INDEX idx_tenant_invitations_token ON tenant_invitations(token);
CREATE INDEX idx_tenant_invitations_email ON tenant_invitations(email);
CREATE INDEX idx_tenant_invitations_tenant ON tenant_invitations(tenant_id);

-- Project indexes
CREATE INDEX idx_projects_tenant ON projects(tenant_id);

-- Flag indexes
CREATE INDEX idx_flags_project ON flags(project_id);
CREATE INDEX idx_flags_rule_logic ON flags(rule_logic);

-- Session indexes
CREATE INDEX idx_session_user_id ON session(user_id);
CREATE INDEX idx_session_token ON session(token);
CREATE INDEX idx_session_expires ON session(expires_at);

-- Account indexes
CREATE INDEX idx_account_user_id ON account(user_id);
CREATE INDEX idx_account_provider ON account(provider_id, account_id);

-- Verification indexes
CREATE INDEX idx_verification_identifier ON verification(identifier);
CREATE INDEX idx_verification_expires ON verification(expires_at);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_members_updated_at BEFORE UPDATE ON tenant_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_flags_updated_at BEFORE UPDATE ON flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_session_updated_at BEFORE UPDATE ON session
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_account_updated_at BEFORE UPDATE ON account
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_verification_updated_at BEFORE UPDATE ON verification
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE users IS 'Core user profiles with Better Auth integration';
COMMENT ON TABLE tenants IS 'Workspaces/Organizations for multi-tenancy';
COMMENT ON TABLE tenant_members IS 'Many-to-many: users can belong to multiple tenants';
COMMENT ON TABLE tenant_invitations IS 'Pending invitations to join tenants';
COMMENT ON TABLE projects IS 'Feature flag projects (scoped to tenant)';
COMMENT ON TABLE flags IS 'Feature flags with rules and rollout configuration';
COMMENT ON TABLE session IS 'Better Auth: User session management';
COMMENT ON TABLE account IS 'Better Auth: OAuth provider accounts and credentials';
COMMENT ON TABLE verification IS 'Better Auth: Magic links, email verification, password reset tokens';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers
DROP TRIGGER IF EXISTS update_verification_updated_at ON verification;
DROP TRIGGER IF EXISTS update_account_updated_at ON account;
DROP TRIGGER IF EXISTS update_session_updated_at ON session;
DROP TRIGGER IF EXISTS update_flags_updated_at ON flags;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_tenant_members_updated_at ON tenant_members;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS jwks CASCADE;
DROP TABLE IF EXISTS verification CASCADE;
DROP TABLE IF EXISTS account CASCADE;
DROP TABLE IF EXISTS session CASCADE;
DROP TABLE IF EXISTS flags CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS tenant_invitations CASCADE;
DROP TABLE IF EXISTS tenant_members CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- +goose StatementEnd
