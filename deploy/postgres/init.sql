-- PostgreSQL schema for WAF configuration
-- Users, custom rules, IP lists, settings

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Users ─────────────────────────────────────────────────────────────────
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,             -- bcrypt hash
    role       TEXT NOT NULL DEFAULT 'viewer',  -- 'admin' | 'editor' | 'viewer'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed a default admin (password: changeme — must be changed on first login)
INSERT INTO users (email, password, role)
VALUES ('admin@waf.local', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQyCamm6VphWFyDCFqXlAXqMu', 'admin')
ON CONFLICT DO NOTHING;

-- ── API keys ──────────────────────────────────────────────────────────────
CREATE TABLE api_keys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    key_hash   TEXT NOT NULL UNIQUE,       -- SHA-256 of the raw key
    last_used  TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Custom rules ──────────────────────────────────────────────────────────
CREATE TABLE custom_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id     INT NOT NULL UNIQUE,       -- SecRule id: 9000000–9999999
    enabled     BOOLEAN NOT NULL DEFAULT true,
    phase       SMALLINT NOT NULL DEFAULT 1,   -- 1–5
    action      TEXT NOT NULL DEFAULT 'deny',  -- 'deny' | 'allow' | 'log'
    variable    TEXT NOT NULL,             -- e.g. 'REQUEST_HEADERS:User-Agent'
    operator    TEXT NOT NULL,             -- e.g. '@contains'
    pattern     TEXT NOT NULL,
    message     TEXT NOT NULL,
    tags        TEXT[] NOT NULL DEFAULT '{}',
    severity    TEXT NOT NULL DEFAULT 'CRITICAL',
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── CRS paranoia / rule toggles ───────────────────────────────────────────
CREATE TABLE crs_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO crs_settings (key, value) VALUES
    ('paranoia_level',      '1'),
    ('anomaly_threshold_in', '5'),
    ('anomaly_threshold_out', '4'),
    ('engine_mode',         'On')
ON CONFLICT (key) DO NOTHING;

-- ── IP allow/deny lists ───────────────────────────────────────────────────
CREATE TABLE ip_lists (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cidr       CIDR NOT NULL,
    list_type  TEXT NOT NULL,             -- 'allow' | 'deny'
    note       TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ip_lists_cidr_type_idx ON ip_lists (cidr, list_type);

-- ── Nginx reload audit trail ──────────────────────────────────────────────
CREATE TABLE reload_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    triggered_by UUID REFERENCES users(id),
    status     TEXT NOT NULL,             -- 'ok' | 'error'
    output     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Helper: auto-update updated_at ───────────────────────────────────────
CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER users_updated_at       BEFORE UPDATE ON users        FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER custom_rules_updated_at BEFORE UPDATE ON custom_rules FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
