-- +goose Up
-- +goose StatementBegin

-- Create users table
CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   TEXT NOT NULL UNIQUE,
    hashed_password         TEXT,
    password_creation_time  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    role                    TEXT NOT NULL,
    status                  TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create users_settings table
CREATE TABLE users_settings (
    id                                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    is_allow_external_registrations       BOOLEAN NOT NULL DEFAULT TRUE,
    is_allow_member_invitations           BOOLEAN NOT NULL DEFAULT TRUE,
    is_member_allowed_to_create_projects  BOOLEAN NOT NULL DEFAULT TRUE
);

-- Create secret_keys table
CREATE TABLE secret_keys (
    secret TEXT PRIMARY KEY
);

-- Create projects table
CREATE TABLE projects (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Security Policies
    is_api_key_required     BOOLEAN NOT NULL DEFAULT FALSE,
    is_filter_by_domain     BOOLEAN NOT NULL DEFAULT FALSE,
    is_filter_by_ip         BOOLEAN NOT NULL DEFAULT FALSE,
    allowed_domains_raw     TEXT NOT NULL DEFAULT '',
    allowed_ips_raw         TEXT NOT NULL DEFAULT '',
    
    -- Rate Limiting & Quotas
    logs_per_second_limit   INTEGER NOT NULL DEFAULT 0,
    max_logs_amount         BIGINT NOT NULL DEFAULT 0,
    max_logs_size_mb        INTEGER NOT NULL DEFAULT 0,
    max_logs_life_days      INTEGER NOT NULL DEFAULT 0,
    max_log_size_kb         INTEGER NOT NULL DEFAULT 0
);

-- Create project_memberships table
CREATE TABLE project_memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    project_id UUID NOT NULL,
    role       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create api_keys table
CREATE TABLE api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    project_id   UUID NOT NULL,
    token_prefix TEXT NOT NULL,
    token_hash   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create audit_logs table
CREATE TABLE audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID,
    project_id UUID,
    message    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add foreign key constraints
ALTER TABLE project_memberships
    ADD CONSTRAINT fk_project_memberships_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

ALTER TABLE project_memberships
    ADD CONSTRAINT fk_project_memberships_project_id
    FOREIGN KEY (project_id)
    REFERENCES projects (id)
    ON DELETE CASCADE;

ALTER TABLE api_keys
    ADD CONSTRAINT fk_api_keys_project_id
    FOREIGN KEY (project_id)
    REFERENCES projects (id)
    ON DELETE CASCADE;

ALTER TABLE audit_logs
    ADD CONSTRAINT fk_audit_logs_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE SET NULL;

-- Add unique constraints
ALTER TABLE project_memberships
    ADD CONSTRAINT uk_project_memberships_user_project
    UNIQUE (user_id, project_id);

ALTER TABLE api_keys
    ADD CONSTRAINT uk_api_keys_token_hash
    UNIQUE (token_hash);

-- Create indexes for better performance
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_created_at ON users (created_at);

CREATE INDEX idx_projects_created_at ON projects (created_at DESC);

CREATE INDEX idx_project_memberships_user_id ON project_memberships (user_id);
CREATE INDEX idx_project_memberships_project_id ON project_memberships (project_id);
CREATE INDEX idx_project_memberships_user_project ON project_memberships (user_id, project_id);
CREATE INDEX idx_project_memberships_project_user ON project_memberships (project_id, user_id);
CREATE INDEX idx_project_memberships_created_at ON project_memberships (created_at ASC);

CREATE INDEX idx_api_keys_project_id ON api_keys (project_id);
CREATE INDEX idx_api_keys_token_hash ON api_keys (token_hash);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id, created_at);
CREATE INDEX idx_audit_logs_project_id ON audit_logs (project_id, created_at);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_project_id;
DROP INDEX IF EXISTS idx_audit_logs_user_id;

DROP INDEX IF EXISTS idx_api_keys_token_hash;
DROP INDEX IF EXISTS idx_api_keys_project_id;

DROP INDEX IF EXISTS idx_project_memberships_created_at;
DROP INDEX IF EXISTS idx_project_memberships_project_user;
DROP INDEX IF EXISTS idx_project_memberships_user_project;
DROP INDEX IF EXISTS idx_project_memberships_project_id;
DROP INDEX IF EXISTS idx_project_memberships_user_id;

DROP INDEX IF EXISTS idx_projects_created_at;

DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_email;

-- Drop foreign key constraints
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_user_id;
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS fk_api_keys_project_id;
ALTER TABLE project_memberships DROP CONSTRAINT IF EXISTS fk_project_memberships_project_id;
ALTER TABLE project_memberships DROP CONSTRAINT IF EXISTS fk_project_memberships_user_id;

-- Drop tables
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS project_memberships;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS secret_keys;
DROP TABLE IF EXISTS users_settings;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
