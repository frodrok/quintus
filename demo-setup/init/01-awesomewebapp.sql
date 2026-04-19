-- awesomewebapp example database
-- runs automatically on first postgres startup

CREATE DATABASE awesomewebapp;
\connect awesomewebapp

-- ------------------------------------------------------------
-- Schema
-- ------------------------------------------------------------

CREATE TABLE users (
    id          SERIAL PRIMARY KEY,
    email       TEXT        NOT NULL UNIQUE,
    full_name   TEXT        NOT NULL,
    role        TEXT        NOT NULL DEFAULT 'user'
                            CHECK (role IN ('admin','editor','user')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE organizations (
    id          SERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL UNIQUE,
    plan        TEXT        NOT NULL DEFAULT 'free'
                            CHECK (plan IN ('free','starter','pro','enterprise')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE org_members (
    org_id      INT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'member'
                CHECK (role IN ('owner','admin','member')),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);

CREATE TABLE projects (
    id          SERIAL PRIMARY KEY,
    org_id      INT         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','archived','deleted')),
    owner_id    INT         REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    id                   SERIAL PRIMARY KEY,
    org_id               INT  NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    stripe_sub_id        TEXT UNIQUE,
    plan                 TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active','past_due','canceled','trialing')),
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    INT         REFERENCES users(id),
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    resource_id TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- indexes
CREATE INDEX ON projects(org_id);
CREATE INDEX ON projects(owner_id);
CREATE INDEX ON org_members(user_id);
CREATE INDEX ON audit_log(actor_id);
CREATE INDEX ON audit_log(created_at DESC);
CREATE INDEX ON audit_log(resource, resource_id);

-- ------------------------------------------------------------
-- Seed data
-- ------------------------------------------------------------

INSERT INTO users (email, full_name, role) VALUES
    ('alice@example.com',   'Alice Lindqvist',  'admin'),
    ('bob@example.com',     'Bob Karlsson',     'editor'),
    ('carol@example.com',   'Carol Eriksson',   'user'),
    ('dave@example.com',    'Dave Nilsson',     'user'),
    ('eve@example.com',     'Eve Johansson',    'user');

INSERT INTO organizations (name, slug, plan) VALUES
    ('Acme Corp',     'acme',       'enterprise'),
    ('Startup AB',    'startup-ab', 'pro'),
    ('Free Tier Ltd', 'free-tier',  'free');

INSERT INTO org_members (org_id, user_id, role) VALUES
    (1, 1, 'owner'),
    (1, 2, 'admin'),
    (1, 3, 'member'),
    (2, 4, 'owner'),
    (2, 5, 'member'),
    (3, 3, 'owner');

INSERT INTO projects (org_id, name, status, owner_id) VALUES
    (1, 'Website Redesign',    'active',   1),
    (1, 'API v2',              'active',   2),
    (1, 'Legacy Migration',    'archived', 1),
    (2, 'MVP Launch',          'active',   4),
    (2, 'Analytics Dashboard', 'active',   5),
    (3, 'Personal Site',       'active',   3);

INSERT INTO subscriptions (org_id, stripe_sub_id, plan, status,
                           current_period_start, current_period_end) VALUES
    (1, 'sub_enterprise_acme', 'enterprise', 'active',
        now() - interval '15 days', now() + interval '15 days'),
    (2, 'sub_pro_startup',     'pro',        'active',
        now() - interval '5 days',  now() + interval '25 days'),
    (3, NULL,                  'free',       'active', NULL, NULL);

INSERT INTO audit_log (actor_id, action, resource, resource_id, metadata) VALUES
    (1, 'create', 'project', '1', '{"name":"Website Redesign"}'),
    (1, 'create', 'project', '2', '{"name":"API v2"}'),
    (2, 'update', 'project', '2', '{"field":"status","old":"draft","new":"active"}'),
    (1, 'invite', 'user',    '3', '{"email":"carol@example.com","org_id":1}'),
    (4, 'create', 'project', '4', '{"name":"MVP Launch"}');