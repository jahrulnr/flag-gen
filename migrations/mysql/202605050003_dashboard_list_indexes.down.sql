DROP INDEX IF EXISTS idx_audit_entries_project_entity_created ON audit_entries;
DROP INDEX IF EXISTS idx_audit_entries_project_created_id ON audit_entries;
DROP INDEX IF EXISTS idx_flags_scope_deleted_key ON flags;
DROP INDEX IF EXISTS idx_segments_project_deleted_key ON segments;
DROP INDEX IF EXISTS idx_environments_project_deleted_key ON environments;
DROP INDEX IF EXISTS idx_projects_deleted_key ON projects;

