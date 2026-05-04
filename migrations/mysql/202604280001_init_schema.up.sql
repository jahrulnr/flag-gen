CREATE TABLE IF NOT EXISTS users (
    id CHAR(36) NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret_encrypted VARBINARY(512) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email),
    KEY idx_users_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_sessions (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    session_hash CHAR(64) NOT NULL,
    user_agent VARCHAR(512) NULL,
    ip_address VARCHAR(64) NULL,
    expires_at DATETIME(3) NOT NULL,
    revoked_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_user_sessions_hash (session_hash),
    KEY idx_user_sessions_user_expiry (user_id, expires_at),
    CONSTRAINT fk_user_sessions_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,
    execution_ms INT NOT NULL DEFAULT 0,
    applied_by_app_version VARCHAR(64) NOT NULL,
    applied_at DATETIME(3) NOT NULL,
    dirty BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (version),
    KEY idx_schema_migrations_dirty (dirty),
    KEY idx_schema_migrations_applied_at (applied_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS projects (
    id CHAR(36) NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    created_by CHAR(36) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    active_key VARCHAR(128) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN `key` ELSE NULL END
    ) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_projects_active_key (active_key),
    KEY idx_projects_deleted_at (deleted_at),
    CONSTRAINT fk_projects_created_by
        FOREIGN KEY (created_by) REFERENCES users (id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS environments (
    id CHAR(36) NOT NULL,
    project_id CHAR(36) NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    active_key VARCHAR(128) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN `key` ELSE NULL END
    ) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_environments_project_active_key (project_id, active_key),
    KEY idx_environments_project_deleted (project_id, deleted_at),
    CONSTRAINT fk_environments_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS segments (
    id CHAR(36) NOT NULL,
    project_id CHAR(36) NOT NULL,
    `key` VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    rules_json JSON NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_by CHAR(36) NULL,
    updated_by CHAR(36) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    active_key VARCHAR(128) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN `key` ELSE NULL END
    ) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_segments_project_active_key (project_id, active_key),
    KEY idx_segments_project_deleted (project_id, deleted_at),
    CONSTRAINT fk_segments_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_segments_created_by
        FOREIGN KEY (created_by) REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT fk_segments_updated_by
        FOREIGN KEY (updated_by) REFERENCES users (id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS flags (
    id CHAR(36) NOT NULL,
    project_id CHAR(36) NOT NULL,
    environment_id CHAR(36) NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    client_side BOOLEAN NOT NULL DEFAULT FALSE,
    value_type VARCHAR(32) NOT NULL,
    default_value_json JSON NOT NULL,
    rollout_percentage DECIMAL(5,2) NULL,
    rollout_stickiness VARCHAR(64) NOT NULL DEFAULT 'userId',
    version INT NOT NULL DEFAULT 1,
    created_by CHAR(36) NULL,
    updated_by CHAR(36) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    active_key VARCHAR(255) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN `key` ELSE NULL END
    ) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_flags_scope_active_key (project_id, environment_id, active_key),
    KEY idx_flags_scope_deleted (project_id, environment_id, deleted_at),
    KEY idx_flags_scope_client_deleted (project_id, environment_id, client_side, deleted_at),
    KEY idx_flags_updated_at (updated_at),
    CONSTRAINT fk_flags_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_flags_environment
        FOREIGN KEY (environment_id) REFERENCES environments (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_flags_created_by
        FOREIGN KEY (created_by) REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT fk_flags_updated_by
        FOREIGN KEY (updated_by) REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_flags_value_type
        CHECK (value_type IN ('boolean', 'json')),
    CONSTRAINT chk_flags_rollout_percentage
        CHECK (rollout_percentage IS NULL OR (rollout_percentage >= 0.00 AND rollout_percentage <= 100.00))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS flag_rules (
    id CHAR(36) NOT NULL,
    flag_id CHAR(36) NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    attribute VARCHAR(255) NULL,
    operator VARCHAR(32) NOT NULL,
    comparison_value_json JSON NOT NULL,
    serve_value_json JSON NOT NULL,
    segment_id CHAR(36) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_flag_rules_flag_priority (flag_id, priority),
    KEY idx_flag_rules_flag (flag_id),
    KEY idx_flag_rules_segment (segment_id),
    CONSTRAINT fk_flag_rules_flag
        FOREIGN KEY (flag_id) REFERENCES flags (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_flag_rules_segment
        FOREIGN KEY (segment_id) REFERENCES segments (id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_flag_rules_operator
        CHECK (operator IN ('equals', 'not_equals', 'in', 'not_in'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS flag_versions (
    id CHAR(36) NOT NULL,
    flag_id CHAR(36) NOT NULL,
    project_id CHAR(36) NOT NULL,
    environment_id CHAR(36) NOT NULL,
    version INT NOT NULL,
    change_type VARCHAR(32) NOT NULL,
    snapshot_json JSON NOT NULL,
    created_by CHAR(36) NULL,
    created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_flag_versions_flag_version (flag_id, version),
    KEY idx_flag_versions_scope_created (project_id, environment_id, created_at),
    KEY idx_flag_versions_created_by (created_by, created_at),
    CONSTRAINT fk_flag_versions_flag
        FOREIGN KEY (flag_id) REFERENCES flags (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_flag_versions_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_flag_versions_environment
        FOREIGN KEY (environment_id) REFERENCES environments (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_flag_versions_created_by
        FOREIGN KEY (created_by) REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_flag_versions_change_type
        CHECK (change_type IN ('created', 'updated', 'deleted', 'rollback'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS audit_entries (
    id CHAR(36) NOT NULL,
    project_id CHAR(36) NULL,
    environment_id CHAR(36) NULL,
    entity_type VARCHAR(64) NOT NULL,
    entity_id CHAR(36) NOT NULL,
    action VARCHAR(64) NOT NULL,
    actor_id CHAR(36) NULL,
    correlation_id VARCHAR(128) NULL,
    old_value_json JSON NULL,
    new_value_json JSON NULL,
    created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_audit_entries_scope_created (project_id, environment_id, created_at),
    KEY idx_audit_entries_entity_created (entity_type, entity_id, created_at),
    KEY idx_audit_entries_actor_created (actor_id, created_at),
    KEY idx_audit_entries_correlation (correlation_id),
    CONSTRAINT fk_audit_entries_actor
        FOREIGN KEY (actor_id) REFERENCES users (id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS api_keys (
    id CHAR(36) NOT NULL,
    project_id CHAR(36) NOT NULL,
    environment_id CHAR(36) NULL,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash CHAR(64) NOT NULL,
    key_type VARCHAR(32) NOT NULL,
    scopes_json JSON NOT NULL,
    created_by CHAR(36) NULL,
    last_used_at DATETIME(3) NULL,
    expires_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_api_keys_hash (key_hash),
    KEY idx_api_keys_scope_type_revoked (project_id, environment_id, key_type, revoked_at),
    KEY idx_api_keys_prefix (key_prefix),
    CONSTRAINT fk_api_keys_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_api_keys_environment
        FOREIGN KEY (environment_id) REFERENCES environments (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_api_keys_created_by
        FOREIGN KEY (created_by) REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_api_keys_type
        CHECK (key_type IN ('server', 'broadcast', 'public'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id CHAR(36) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    actor_id CHAR(36) NULL,
    project_id CHAR(36) NULL,
    request_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    response_code INT NULL,
    response_json JSON NULL,
    locked_until DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_idempotency_actor_key (idempotency_key, actor_id),
    KEY idx_idempotency_project_created (project_id, created_at),
    KEY idx_idempotency_status_locked (status, locked_until),
    CONSTRAINT fk_idempotency_actor
        FOREIGN KEY (actor_id) REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT fk_idempotency_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_idempotency_status
        CHECK (status IN ('processing', 'completed', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS outbox_events (
    id CHAR(36) NOT NULL,
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id CHAR(36) NOT NULL,
    aggregate_version INT NOT NULL,
    scope_version INT NOT NULL,
    project_id CHAR(36) NOT NULL,
    environment_id CHAR(36) NULL,
    event_type VARCHAR(128) NOT NULL,
    payload_json JSON NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    next_attempt_at DATETIME(3) NULL,
    locked_by VARCHAR(128) NULL,
    locked_at DATETIME(3) NULL,
    last_error TEXT NULL,
    created_at DATETIME(3) NOT NULL,
    published_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_outbox_status_attempt_created (status, next_attempt_at, created_at),
    KEY idx_outbox_aggregate_version (aggregate_type, aggregate_id, aggregate_version),
    KEY idx_outbox_scope_created (project_id, environment_id, created_at),
    CONSTRAINT fk_outbox_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_outbox_environment
        FOREIGN KEY (environment_id) REFERENCES environments (id)
        ON DELETE CASCADE,
    CONSTRAINT chk_outbox_status
        CHECK (status IN ('pending', 'publishing', 'published', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS snapshot_metadata (
    id CHAR(36) NOT NULL,
    project_id CHAR(36) NOT NULL,
    environment_id CHAR(36) NOT NULL,
    scope_version INT NOT NULL,
    checksum CHAR(64) NOT NULL,
    file_path VARCHAR(1024) NOT NULL,
    source_outbox_event_id CHAR(36) NULL,
    generated_at DATETIME(3) NOT NULL,
    last_validated_at DATETIME(3) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'valid',
    PRIMARY KEY (id),
    UNIQUE KEY uq_snapshot_metadata_scope (project_id, environment_id),
    KEY idx_snapshot_metadata_status_generated (status, generated_at),
    KEY idx_snapshot_metadata_source_outbox (source_outbox_event_id),
    CONSTRAINT fk_snapshot_metadata_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_snapshot_metadata_environment
        FOREIGN KEY (environment_id) REFERENCES environments (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_snapshot_metadata_outbox
        FOREIGN KEY (source_outbox_event_id) REFERENCES outbox_events (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_snapshot_metadata_status
        CHECK (status IN ('valid', 'invalid'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
