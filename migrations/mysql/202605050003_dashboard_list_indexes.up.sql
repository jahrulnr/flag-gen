CREATE INDEX IF NOT EXISTS idx_projects_deleted_key ON projects (deleted_at, `key`);

CREATE INDEX IF NOT EXISTS idx_environments_project_deleted_key ON environments (project_id, deleted_at, `key`);

CREATE INDEX IF NOT EXISTS idx_segments_project_deleted_key ON segments (project_id, deleted_at, `key`);

CREATE INDEX IF NOT EXISTS idx_flags_scope_deleted_key ON flags (project_id, environment_id, deleted_at, `key`);

CREATE INDEX IF NOT EXISTS idx_audit_entries_project_created_id ON audit_entries (project_id, created_at, id);

CREATE INDEX IF NOT EXISTS idx_audit_entries_project_entity_created ON audit_entries (project_id, entity_type, entity_id, created_at);

