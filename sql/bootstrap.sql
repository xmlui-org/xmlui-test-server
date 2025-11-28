-- bootstrap.sql
-- Purpose: create starter tables and add demo data, idempotently.
-- Style: beginner-friendly; uses INSERT OR IGNORE everywhere.

PRAGMA foreign_keys = ON;   -- enforce FKs during setup

BEGIN;

-- =========================
-- USERS
-- =========================
CREATE TABLE IF NOT EXISTS users (
id          INTEGER PRIMARY KEY AUTOINCREMENT,
email       TEXT NOT NULL UNIQUE,        -- natural key
name        TEXT NOT NULL,
created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Demo users (OR IGNORE = skip if already present)
INSERT OR IGNORE INTO users (email, name) VALUES
('alice@example.com', 'Alice Carter'),
('bob@example.com',   'Bob Nguyen'),
('carol@example.com', 'Carol Diaz');

-- =========================
-- PROJECTS  (owned by a user)
-- =========================
CREATE TABLE IF NOT EXISTS projects (
id          INTEGER PRIMARY KEY AUTOINCREMENT,
owner_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
name        TEXT NOT NULL,
description TEXT,
status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
UNIQUE (owner_id, name)                     -- same owner can't reuse the same name
);

CREATE INDEX IF NOT EXISTS projects_owner_idx  ON projects(owner_id);
CREATE INDEX IF NOT EXISTS projects_status_idx ON projects(status);

-- Demo projects (owner looked up by email)
INSERT OR IGNORE INTO projects (owner_id, name, description, status)
SELECT
(SELECT id FROM users WHERE email = 'alice@example.com'),
'Website Redesign',
'Revamp marketing site for Q4',
'active';

INSERT OR IGNORE INTO projects (owner_id, name, description, status)
SELECT
(SELECT id FROM users WHERE email = 'bob@example.com'),
'Mobile App MVP',
'First-cut iOS/Android client',
'active';

-- =========================
-- TASKS  (belong to a project, optional assignee)
-- =========================
CREATE TABLE IF NOT EXISTS tasks (
id           INTEGER PRIMARY KEY AUTOINCREMENT,
project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
assignee_id  INTEGER REFERENCES users(id) ON DELETE SET NULL,
title        TEXT NOT NULL,
details      TEXT,
status       TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo','doing','done')),
priority     INTEGER NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5),
due_date     TEXT,
created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
UNIQUE (project_id, title)                  -- titles unique within a project
);

CREATE INDEX IF NOT EXISTS tasks_project_idx  ON tasks(project_id);
CREATE INDEX IF NOT EXISTS tasks_assignee_idx ON tasks(assignee_id);
CREATE INDEX IF NOT EXISTS tasks_status_idx   ON tasks(status);
CREATE INDEX IF NOT EXISTS tasks_priority_idx ON tasks(priority);

-- Demo tasks for "Website Redesign"
INSERT OR IGNORE INTO tasks (project_id, assignee_id, title, details, status, priority, due_date)
SELECT
(SELECT p.id
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE u.email = 'alice@example.com' AND p.name = 'Website Redesign'),
(SELECT id FROM users WHERE email = 'alice@example.com'),
'Audit current site',
'Inventory pages, assets, SEO, analytics, lighthouse scores.',
'doing',
2,
'2025-10-01';

INSERT OR IGNORE INTO tasks (project_id, assignee_id, title, details, status, priority, due_date)
SELECT
(SELECT p.id
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE u.email = 'alice@example.com' AND p.name = 'Website Redesign'),
(SELECT id FROM users WHERE email = 'bob@example.com'),
'Design homepage',
'Wireframe + high-fidelity mockups for new homepage.',
'todo',
4,
'2025-10-10';

INSERT OR IGNORE INTO tasks (project_id, assignee_id, title, details, status, priority, due_date)
SELECT
(SELECT p.id
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE u.email = 'alice@example.com' AND p.name = 'Website Redesign'),
(SELECT id FROM users WHERE email = 'carol@example.com'),
'Implement responsive layout',
'Grid/flexbox for breakpoints 360/768/1024/1440.',
'todo',
3,
'2025-10-15';

-- Demo tasks for "Mobile App MVP"
INSERT OR IGNORE INTO tasks (project_id, assignee_id, title, details, status, priority, due_date)
SELECT
(SELECT p.id
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE u.email = 'bob@example.com' AND p.name = 'Mobile App MVP'),
(SELECT id FROM users WHERE email = 'bob@example.com'),
'Auth flow',
'Email/password + OAuth; persist token; logout; forgot password.',
'doing',
5,
'2025-10-05';

INSERT OR IGNORE INTO tasks (project_id, assignee_id, title, details, status, priority, due_date)
SELECT
(SELECT p.id
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE u.email = 'bob@example.com' AND p.name = 'Mobile App MVP'),
(SELECT id FROM users WHERE email = 'alice@example.com'),
'API client',
'Typed client, retries/backoff, request timeout, error normalization.',
'todo',
4,
'2025-10-08';

INSERT OR IGNORE INTO tasks (project_id, assignee_id, title, details, status, priority, due_date)
SELECT
(SELECT p.id
FROM projects p
JOIN users u ON u.id = p.owner_id
WHERE u.email = 'bob@example.com' AND p.name = 'Mobile App MVP'),
(SELECT id FROM users WHERE email = 'carol@example.com'),
'Push notifications',
'Baseline via FCM/APNs; categories; opt-in UX.',
'todo',
3,
'2025-10-18';

COMMIT;

PRAGMA foreign_keys = OFF;  -- back to your desired default for this connection

-- (Optional) quick checks when run in the sqlite3 CLI:
-- SELECT count(*) AS users    FROM users;
-- SELECT count(*) AS projects FROM projects;
-- SELECT count(*) AS tasks    FROM tasks;
