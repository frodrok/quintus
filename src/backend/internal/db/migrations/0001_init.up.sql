CREATE TABLE connections (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name                 TEXT NOT NULL,
  driver               TEXT NOT NULL,
  dsn_encrypted        BYTEA NOT NULL,
  read_only            BOOLEAN NOT NULL DEFAULT true,
  statement_timeout_ms INT NOT NULL DEFAULT 30000,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by_sub       TEXT,
  created_by_email     TEXT
);

CREATE TABLE queries (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name           TEXT NOT NULL,
  description    TEXT,
  connection_id  UUID NOT NULL REFERENCES connections(id),
  sql            TEXT NOT NULL,
  parameters     JSONB NOT NULL DEFAULT '[]',
  column_masks   JSONB NOT NULL DEFAULT '[]',
  owner_sub      TEXT NOT NULL,
  owner_email    TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE runs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_sub        TEXT NOT NULL,
  user_email      TEXT NOT NULL,
  user_groups     TEXT NOT NULL,
  user_role       TEXT NOT NULL,
  connection_id   UUID NOT NULL REFERENCES connections(id),
  query_id        UUID REFERENCES queries(id),
  sql             TEXT NOT NULL,
  parameters      JSONB,
  export_format   TEXT,
  masked_columns  JSONB NOT NULL DEFAULT '[]',
  started_at      TIMESTAMPTZ NOT NULL,
  finished_at     TIMESTAMPTZ,
  duration_ms     INT,
  row_count       INT,
  bytes_returned  BIGINT,
  status          TEXT NOT NULL,
  error_message   TEXT,
  client_ip       INET,
  user_agent      TEXT
);

CREATE INDEX runs_user_started  ON runs (user_sub, started_at DESC);
CREATE INDEX runs_query_started ON runs (query_id, started_at DESC);
CREATE INDEX runs_started_at    ON runs (started_at DESC);
CREATE INDEX runs_export_format ON runs (export_format) WHERE export_format IS NOT NULL;