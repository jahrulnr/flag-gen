ALTER TABLE users DROP FOREIGN KEY fk_users_role;
ALTER TABLE users DROP KEY idx_users_role;
ALTER TABLE users DROP COLUMN role_id;
DROP TABLE IF EXISTS roles;
