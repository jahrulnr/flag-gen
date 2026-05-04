-- One-time dev bootstrap: first user only (empty users table). Login email admin@local, password admin.
-- Skipped automatically if any user already exists (e.g. POST /v1/system/install ran first).
INSERT INTO users (id, email, name, password_hash, two_factor_enabled, status, created_at, updated_at)
SELECT 'f0000000-0000-4000-8000-000000000001',
       'admin@local',
       'admin',
       '$2a$10$pBubDYB9uj5t62QhZNZ6M.Q3hxxT.EfVTLewB9krY0.Ppx3EXWcgm',
       FALSE,
       'active',
       UTC_TIMESTAMP(3),
       UTC_TIMESTAMP(3)
WHERE NOT EXISTS (SELECT 1 FROM users LIMIT 1);
