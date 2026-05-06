CREATE TABLE IF NOT EXISTS roles (
    id CHAR(36) NOT NULL,
    `key` VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NULL,
    permissions_json JSON NOT NULL,
    system_role BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_roles_key (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO roles (id, `key`, name, description, permissions_json, system_role, created_at, updated_at)
VALUES
  ('f1000000-0000-4000-8000-000000000001', 'readonly', 'Read only', 'Can view dashboard data but cannot mutate workspace resources.', JSON_ARRAY('*:read'), TRUE, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  ('f1000000-0000-4000-8000-000000000002', 'editor', 'Editor', 'Can read and mutate workspace resources, excluding administration pages.', JSON_ARRAY('*:read', '*:write'), TRUE, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  ('f1000000-0000-4000-8000-000000000003', 'business', 'Business', 'Can read operational data and evaluate flags for business validation.', JSON_ARRAY('*:read', 'flags:evaluate'), TRUE, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  ('f1000000-0000-4000-8000-000000000004', 'admin', 'Admin', 'Can manage workspace resources, API keys, users, and roles.', JSON_ARRAY('*:read', '*:write', 'admin:*'), TRUE, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  ('f1000000-0000-4000-8000-000000000005', 'superadmin', 'Superadmin', 'Unrestricted administrative access.', JSON_ARRAY('*'), TRUE, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3))
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  permissions_json = VALUES(permissions_json),
  system_role = VALUES(system_role),
  updated_at = UTC_TIMESTAMP(3);

ALTER TABLE users
    ADD COLUMN role_id CHAR(36) NULL AFTER status;

UPDATE users
SET role_id = 'f1000000-0000-4000-8000-000000000004'
WHERE role_id IS NULL;

ALTER TABLE users
    MODIFY role_id CHAR(36) NOT NULL,
    ADD KEY idx_users_role (role_id),
    ADD CONSTRAINT fk_users_role
        FOREIGN KEY (role_id) REFERENCES roles (id)
        ON DELETE RESTRICT;
