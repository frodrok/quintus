# QueryExplorer — Specification & Task Breakdown

**Project codename:** Quintus

An internal web tool for analysts and ops to run saved and ad-hoc SQL queries
against configured databases. Authentication is handled upstream by Traefik via
OIDC forward-auth; the app trusts signed headers for identity and group
membership. Every query execution — including exports — is recorded as an
immutable run, with optional group-gated column masking for sensitive fields.

---

## 1. Goals

- **Stateless identity.** Traefik + a forward-auth proxy (oauth2-proxy or equivalent) handle the OIDC dance. QueryExplorer reads identity from trusted request headers on every request.
- A library of **saved queries** with typed, named parameters.
- Ad-hoc SQL execution for authorized groups.
- Results displayed as a virtualized table, exportable to CSV and XLSX — **exports are themselves runs**, not a separate object.
- **Group-gated column masking**: sensitive columns can be declared per query and revealed only to users in permitted groups; masking decisions are recorded on every run.
- Immutable audit log: every run records the identity (email + subject), groups at execution time, SQL, parameter values, masking applied, timing, row/byte counts, and status.
- Self-hostable, single binary, minimal operational surface.

## 2. Non-Goals (v1)

- Dashboards, charts, scheduled queries, alerts.
- Embedding / public share links.
- Result caching.
- In-app user management — there is no users table. Identity and roles are derived from headers on every request.
- Write queries against source data by default (read-only; writes opt-in per connection).
- OIDC inside the app — that is Traefik's job.

## 3. Authentication & Authorization Model

### Trust boundary

```
Internet ──► Traefik ──► forward-auth (oauth2-proxy) ──► IdP (Keycloak / Azure AD / ...)
                │
                │  On success, Traefik attaches:
                │    X-User-Sub
                │    X-User-Email
                │    X-User-Name
                │    X-User-Groups     (comma-separated)
                │    X-Auth-Proxy-Secret   (shared secret)
                ▼
           QueryExplorer backend
```

The backend **never** talks to the IdP directly. It trusts headers only when:

1. The request carries a valid `X-Auth-Proxy-Secret` matching the configured value, **and**
2. A Kubernetes NetworkPolicy restricts inbound traffic to the Traefik pod.

At ingress, Traefik **strips any inbound `X-User-*` and `X-Auth-Proxy-Secret` headers** before forward-auth runs, so clients cannot preset them.

### Roles from groups (stateless)

There is no users table and no persisted role state. Roles are derived per-request from the `X-User-Groups` header via configuration:

```
QE_ROLE_MAPPING=admin:qe-admins,editor:qe-editors,viewer:qe-viewers
```

A user's effective role is the highest-privilege role any of their groups maps to. If no group maps, the user is anonymous and receives 403 on `/api/*`.

| Role   | Capabilities                                                                            |
|--------|-----------------------------------------------------------------------------------------|
| viewer | Run saved queries, view own runs, export rows up to `viewer_export_cap`                 |
| editor | All viewer rights + create/edit saved queries, run ad-hoc SQL, higher export cap        |
| admin  | All editor rights + manage connections, view all audit data, unlimited exports          |

### Capability groups (separate dimensions)

Two capabilities sit **orthogonal** to role, gated by group membership:

- **PII access** — controls whether masked columns are revealed. Declared via `QE_PII_GROUPS`.
- **Ad-hoc SQL** — controls whether the user can execute arbitrary SQL (saved queries are always available to the appropriate role). Declared via `QE_ADHOC_GROUPS`.

A user can be an `editor` without being in either group — they can author saved queries with declared column masks, but cannot write raw SQL or see masked columns. This makes ad-hoc a deliberate privilege, not a side-effect of being an editor.

```
QE_PII_GROUPS=pii-approved,pii-support-partial
QE_ADHOC_GROUPS=qe-adhoc
```

Role capability matrix:

| Capability                  | Required role   | Additional group required |
|-----------------------------|-----------------|---------------------------|
| Run saved queries           | viewer          | —                         |
| View own run history        | viewer          | —                         |
| See masked column values    | viewer          | PII group                 |
| Create / edit saved queries | editor          | —                         |
| Run ad-hoc SQL              | editor          | ad-hoc group              |
| Manage connections          | admin           | —                         |
| View full audit log         | admin           | —                         |

### Identity on runs

Because there is no users table, each `runs` row stores:

- `user_sub` — stable IdP subject (what you join on long-term)
- `user_email` — convenience for display
- `user_groups` — full comma-separated list **at the time of execution**

This is deliberate. If someone is later removed from `pii-approved`, the historical run still shows they had that access when they ran the query.

## 4. Architecture

- **Backend**: Go single binary, `chi` router, `pgx` for the app database, `database/sql` with driver plugins for target databases.
- **Frontend**: Vue 3 + Vuetify, Monaco editor for SQL, AG Grid Community for results; built and embedded into the Go binary via `embed.FS`.
- **App database**: PostgreSQL (separate from any target database).
- **Secrets**: Connection DSNs encrypted with AES-256-GCM; key from env or KMS.
- **Deployment**: Single container, behind Traefik IngressRoute with ForwardAuth middleware.

### Request flow

```
Browser ──► Traefik
              │  (forward-auth: unauthenticated → IdP flow)
              │  (authenticated: headers injected, request proxied)
              ▼
          QueryExplorer
              ├─► /api/queries           list / create / update saved queries
              ├─► /api/connections       admin only
              ├─► /api/runs              execute saved or ad-hoc SQL; export = run with format
              ├─► /api/runs/{id}         status + preview
              ├─► /api/runs/{id}/stream  stream CSV or XLSX
              └─► /api/audit/runs        admin: query the runs log
```

## 5. Core Data Model

```sql
-- connections: target databases users can query
CREATE TABLE connections (
  id                    UUID PRIMARY KEY,
  name                  TEXT NOT NULL,
  driver                TEXT NOT NULL,            -- postgres | mysql | ...
  dsn_encrypted         BYTEA NOT NULL,           -- AES-256-GCM
  read_only             BOOLEAN NOT NULL DEFAULT true,
  statement_timeout_ms  INT NOT NULL DEFAULT 30000,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by_sub        TEXT,                     -- X-User-Sub at creation time
  created_by_email      TEXT
);

-- queries: saved, parameterized SQL with optional column masking rules
CREATE TABLE queries (
  id             UUID PRIMARY KEY,
  name           TEXT NOT NULL,
  description    TEXT,
  connection_id  UUID NOT NULL REFERENCES connections(id),
  sql            TEXT NOT NULL,
  parameters     JSONB NOT NULL DEFAULT '[]',
  column_masks   JSONB NOT NULL DEFAULT '[]',     -- see section 7.3
  owner_sub      TEXT NOT NULL,
  owner_email    TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- runs: every execution, saved or ad-hoc, preview or export
CREATE TABLE runs (
  id              UUID PRIMARY KEY,
  user_sub        TEXT NOT NULL,                  -- from X-User-Sub
  user_email      TEXT NOT NULL,                  -- from X-User-Email
  user_groups     TEXT NOT NULL,                  -- from X-User-Groups, CSV, as-received
  user_role       TEXT NOT NULL,                  -- derived role at execution time
  connection_id   UUID NOT NULL REFERENCES connections(id),
  query_id        UUID REFERENCES queries(id),    -- null for ad-hoc
  sql             TEXT NOT NULL,                  -- final SQL sent to driver
  parameters      JSONB,
  export_format   TEXT,                           -- null (preview) | csv | xlsx
  masked_columns  JSONB NOT NULL DEFAULT '[]',    -- columns masked for this run
  started_at      TIMESTAMPTZ NOT NULL,
  finished_at     TIMESTAMPTZ,
  duration_ms     INT,
  row_count       INT,
  bytes_returned  BIGINT,
  status          TEXT NOT NULL,                  -- running | success | error | cancelled
  error_message   TEXT,
  client_ip       INET,
  user_agent      TEXT
);

CREATE INDEX runs_user_started    ON runs (user_sub, started_at DESC);
CREATE INDEX runs_query_started   ON runs (query_id, started_at DESC);
CREATE INDEX runs_started_at      ON runs (started_at DESC);
CREATE INDEX runs_export_format   ON runs (export_format) WHERE export_format IS NOT NULL;
```

**Note on exports.** There is no separate exports table. A download is a run where `export_format` is `csv` or `xlsx`. This gives one unified audit timeline and one set of concurrency/rate-limit primitives.

## 6. Authentication Middleware

Every `/api/*` request passes through an `identity` middleware that:

1. **Verifies the proxy-secret header** matches `QE_PROXY_SHARED_SECRET`. If missing or mismatched → 401, log at warn level.
2. Reads `X-User-Sub`, `X-User-Email`, `X-User-Name`, `X-User-Groups`. If `X-User-Sub` or `X-User-Email` is missing → 401.
3. Parses groups (comma-separated), computes effective role via `QE_ROLE_MAPPING`. If no role matches → 403.
4. Injects an `Identity` struct into the request context containing `Sub`, `Email`, `Name`, `Groups []string`, `Role`.

No session, no cookies, no token verification. The backend is stateless w.r.t. auth.

### Logout

The app exposes `/api/logout` that returns a redirect URL pointing at the configured `QE_LOGOUT_URL` (e.g., oauth2-proxy's `/oauth2/sign_out?rd=...`). The frontend triggers navigation.

### Configuration

```
QE_PROXY_SHARED_SECRET=<random 32+ bytes>
QE_HEADER_SUB=X-User-Sub
QE_HEADER_EMAIL=X-User-Email
QE_HEADER_NAME=X-User-Name
QE_HEADER_GROUPS=X-User-Groups
QE_GROUP_SEPARATOR=,
QE_ROLE_MAPPING=admin:qe-admins,editor:qe-editors,viewer:qe-viewers
QE_PII_GROUPS=pii-approved,pii-support-partial
QE_LOGOUT_URL=https://queryexplorer.example.com/oauth2/sign_out?rd=%2F
```

## 7. Query Execution

### 7.1 Saved queries

Parameters declared as JSONB:

```json
[
  {"name": "start_date", "type": "date", "required": true},
  {"name": "region", "type": "enum", "values": ["EU", "US", "APAC"], "default": "EU"},
  {"name": "limit", "type": "int", "default": 100}
]
```

SQL uses `:name` placeholders. Executor rewrites to driver-native placeholders via `sqlx.Named` + `sqlx.Rebind`. **Values always passed separately — no string interpolation.**

### 7.2 Ad-hoc SQL

- No parameter binding.
- Parser enforces statement type against `connections.read_only`:
  - Read-only: only `SELECT`, `WITH ... SELECT`, `SHOW`, `EXPLAIN`.
  - Writable: any statement but requires `editor` or `admin` role.
- Use `vitess.io/vitess/go/vt/sqlparser` or equivalent. Multi-statement input rejected.
- Ad-hoc SQL cannot use column masking (masking requires a saved query definition). If ad-hoc is allowed against a PII-bearing connection, all rows return unmasked — so `QE_PII_GROUPS` membership is enforced on the **connection** for ad-hoc (configurable flag `connections.adhoc_requires_pii_group`).

### 7.3 Column masking

Per-query declaration:

```json
"column_masks": [
  {
    "column": "personnummer",
    "visible_to_groups": ["pii-approved"],
    "mask": "redacted"
  },
  {
    "column": "email",
    "visible_to_groups": ["pii-approved", "pii-support-partial"],
    "mask": "partial"
  }
]
```

Mask functions:

| Mask       | Behaviour                                                       |
|------------|-----------------------------------------------------------------|
| `redacted` | Replace value with `***REDACTED***` (length fixed)              |
| `partial`  | Type-specific: email → `f***@example.com`; string → first char + `***` + last char; number → `****` |
| `null`     | Replace with NULL                                               |
| `hash`     | SHA-256 of value (stable across runs; useful for deduping reports without revealing value) |

**Enforcement point.** Masking is applied in the streaming pipeline between the driver result set and the serializer (JSON for preview, CSV writer, XLSX StreamWriter). Column matching is by **result-set column name**. Aliases bypass this by design — the query author is responsible; the save-time linter warns if a saved query aliases a declared masked column.

**Audit.** Every run records `masked_columns` — the subset actually masked for this execution based on the user's groups at the time.

### 7.4 Execution pipeline

1. Resolve identity + role from middleware.
2. Validate: role permitted for action, connection permitted, parameters valid, SQL passes read-only check.
3. Compute masked columns for this user's groups × query's column_masks.
4. Insert `runs` row with `status = 'running'`, `started_at = now()`, `masked_columns` captured.
5. Open pooled connection with `context.WithTimeout(ctx, connection.statement_timeout_ms)`.
6. `db.QueryContext(ctx, rewritten_sql, args...)`.
7. Stream rows through mask pipeline → serializer; track `row_count` and `bytes_returned`.
8. On completion (success/error/cancel): update run row.
9. On HTTP context cancel (client disconnect): DB context cancels, driver issues backend cancel (e.g. `pg_cancel_backend`), run marked `cancelled`.

### 7.5 Concurrency & rate limits

- Max concurrent runs per user (`user_sub`): 5.
- Max concurrent runs per connection: 20.
- 429 when exceeded; no silent queueing.

## 8. Results & Exports

Remember: **an export is a run.** The UI distinguishes "preview" (no `export_format`) from "export" (`csv` or `xlsx`), but both go through the same pipeline and both write a `runs` row.

### 8.1 Preview

- `POST /api/runs` with `{connection_id, sql, parameters}` or `{query_id, parameters}`.
- Response streams JSON rows up to `max_ui_rows` (default 10 000).
- Returns `run_id`, `truncated` flag, column metadata.

### 8.2 Export

- `POST /api/runs` with `export_format: "csv"` or `"xlsx"`.
- Response is the streamed file directly, with `Content-Disposition: attachment`.
- CSV: `encoding/csv` + `http.Flusher`, UTF-8 BOM, RFC 4180 quoting.
- XLSX: `xuri/excelize` `StreamWriter`, single sheet `Results`.
- Per-role export row caps:
  - `viewer`: 10 000
  - `editor`: 100 000
  - `admin`: unlimited (bounded by connection timeout)

### 8.3 Filename convention

```
{query_name_slug | "adhoc"}-{run_id_short}.{csv|xlsx}
```

## 9. Audit Log

The `runs` table **is** the audit log. No separate system, no separate exports table.

### Guarantees

- Every run inserts its row *before* the target database is contacted, so crashes and hangs are visible.
- Identity (`user_sub`, `user_email`, `user_groups`, `user_role`) is captured **at execution time** and never mutated afterwards.
- `masked_columns` records exactly what was hidden for this run.
- Rows are insert-only from application code; schema grants enforce this (app role has `INSERT, SELECT, UPDATE(finished_at, duration_ms, row_count, bytes_returned, status, error_message)` on `runs`, nothing else).

### Admin UI

- `/admin/audit` — filter by user email/sub, connection, query, status, export format, group, date range.
- Columns: started_at, user, role, connection, query name or "Ad-hoc", format (preview/csv/xlsx), duration, rows, bytes, status, masked columns count.
- Drill-down: full SQL, parameter values, full group list at time of run, masked columns detail, error.
- "Export this audit view" is itself a run with `export_format` set.

### Retention

- Default: retain indefinitely.
- Optional: nightly archive of runs older than N days to S3-compatible storage (compressed JSONL or Parquet); deletion from primary table only after archive verification.

## 10. Security Requirements

- All `/api/*` routes behind the identity middleware with proxy-secret enforcement.
- Inbound `X-User-*` and proxy-secret headers **stripped at Traefik** before forward-auth runs.
- Kubernetes NetworkPolicy: backend pod accepts traffic only from Traefik.
- CSRF token on state-changing methods (header-based, since there's no cookie session). Token delivered as a header the SPA reads from a GET endpoint and echoes back.
- Connection DSNs encrypted at rest (AES-256-GCM); key sourced from env or KMS.
- No query result written to server disk; streaming end-to-end.
- TLS required; HSTS, secure headers, CSP with frame-ancestors deny.
- Structured JSON app logs never include row values, parameter values of masked columns, or the `X-Auth-Proxy-Secret`. Query content lives in `runs` only.

## 11. Operational Requirements

- Health: `/healthz` (liveness), `/readyz` (checks app DB reachability + at least one connection usable).
- Prometheus `/metrics`: run counts by status and format, run duration histogram, active runs gauge, export bytes counter, mask applications counter (by group).
- Graceful shutdown: drain in-flight runs up to configurable timeout, then cancel.
- Config via env vars only; no config file required.
- Single container image (non-root), arm64 + amd64.

---

# Task Breakdown

Tasks grouped by milestone. Each task is roughly one focused work session.

## Milestone 1 — Foundations

- [ ] **M1.1** Repo scaffold: Go module, `chi` router, `/healthz`, `slog` structured logging, env config loader.
- [ ] **M1.2** App database setup: `pgx` pool, `golang-migrate`, initial migration with `connections`, `queries`, `runs`.
- [ ] **M1.3** Frontend scaffold: Vue 3 + Vite + Vuetify, router, API client, build output embedded via `embed.FS`.
- [ ] **M1.4** Docker image: multi-stage build, non-root, arm64 + amd64, single final image.
- [ ] **M1.5** Makefile / task runner for dev loop (`make run`, `make test`, `make migrate`).

## Milestone 2 — Identity Middleware

- [ ] **M2.1** Config loader for `QE_PROXY_SHARED_SECRET`, `QE_HEADER_*`, `QE_ROLE_MAPPING`, `QE_PII_GROUPS`; fail-fast if unset.
- [ ] **M2.2** Identity middleware: verify proxy secret (constant-time compare), parse headers, compute role, inject `Identity` into context.
- [ ] **M2.3** `RequireRole(role)` and `RequireGroup(group)` helpers for handler wrapping.
- [ ] **M2.4** CSRF token endpoint + header-based verification on state-changing methods.
- [ ] **M2.5** `/api/me` — returns current identity + role + PII group membership (for the frontend to shape the UI).
- [ ] **M2.6** `/api/logout` — returns `{url: QE_LOGOUT_URL}`.
- [ ] **M2.7** Tests: spoofed headers without proxy secret → 401; missing sub → 401; unmapped groups → 403; valid admin group → 200.

## Milestone 3 — Connections

- [ ] **M3.1** AES-256-GCM helper; key from env (`QE_DSN_ENCRYPTION_KEY`); test vectors.
- [ ] **M3.2** Connection CRUD (admin): create, list, update, delete. DSN never returned in responses.
- [ ] **M3.3** `POST /api/connections/{id}/test` — opens pooled connection, runs `SELECT 1` with short timeout.
- [ ] **M3.4** Pool registry: per-connection `*sql.DB` with `SetMaxOpenConns`, reused across requests; reload on update.
- [ ] **M3.5** Driver plugin interface; PostgreSQL driver first.
- [ ] **M3.6** Admin UI: connections list, form (name, driver, DSN, read-only, timeout, `adhoc_requires_pii_group` flag), test button.

## Milestone 4 — Query Execution Core

- [ ] **M4.1** Parameter schema validation on save and on execute; coerce types.
- [ ] **M4.2** Parameter binding: `:name` → driver-native placeholders via `sqlx.Named` + `Rebind`.
- [ ] **M4.3** Read-only SQL enforcement via parser; multi-statement rejection.
- [ ] **M4.4** Run insertion before execution (`status = 'running'`), terminal update on completion.
- [ ] **M4.5** Context-based cancellation wired from HTTP request to driver; verify cancel on client disconnect.
- [ ] **M4.6** Row + byte counting during streaming; truncation flag for preview.
- [ ] **M4.7** Per-user and per-connection concurrency limits via `golang.org/x/sync/semaphore`.
- [ ] **M4.8** `POST /api/runs` — handles preview and export based on `export_format`; routes to CSV/XLSX writer when set.
- [ ] **M4.9** `GET /api/runs/{id}` — run status and metadata (for polling).

## Milestone 5 — Column Masking

- [ ] **M5.1** Mask functions: `redacted`, `partial`, `null`, `hash`. Type-aware; unit tests.
- [ ] **M5.2** Mask pipeline: per-column decision computed once per run, applied row-by-row in streaming serializer.
- [ ] **M5.3** `column_masks` JSONB validation on query save: schema check, column name linting, alias warning.
- [ ] **M5.4** `masked_columns` written to run row based on user's groups × query definition.
- [ ] **M5.5** Tests: three fake users (admin+PII, editor+PII, editor-no-PII) running the same query must produce three different masking outcomes and three matching audit rows.

## Milestone 6 — Saved Queries UI

- [ ] **M6.1** Query CRUD endpoints (editor+ to create/update; viewer+ to read).
- [ ] **M6.2** Query list view: filter by owner, connection, search by name.
- [ ] **M6.3** Query editor page: Monaco SQL editor, connection selector, parameter definition UI, column-mask definition UI, run button.
- [ ] **M6.4** Parameter input form generated from schema: date picker, enum select, number input, required markers.
- [ ] **M6.5** Results panel: AG Grid with virtualized rows, column type hints, truncation banner, masked-column indicator in header.
- [ ] **M6.6** Export buttons trigger a second run with `export_format` set; UI shows the new run in run history.

## Milestone 7 — Audit & Admin

- [ ] **M7.1** `GET /api/audit/runs` (admin): filter by user_sub/email, connection, query, status, format, group-contains, date range; pagination.
- [ ] **M7.2** `GET /api/audit/runs/{id}` — full run detail including SQL, parameters, groups snapshot, masked columns.
- [ ] **M7.3** Admin UI: audit table with filters; drill-down modal; "Export audit view" which itself writes a run.
- [ ] **M7.4** Database grants verification test: app role cannot UPDATE identity fields or DELETE runs.

## Milestone 8 — Hardening

- [ ] **M8.1** Secure headers middleware: HSTS, CSP, frame-ancestors, X-Content-Type-Options.
- [ ] **M8.2** Prometheus metrics; dashboards in JSON for Grafana import.
- [ ] **M8.3** Graceful shutdown with in-flight run drain.
- [ ] **M8.4** Integration tests: spin up app DB + target Postgres + fake Traefik (header-injecting reverse proxy) + run full scenarios.
- [ ] **M8.5** Load test: 50 concurrent users, verify concurrency caps and metric accuracy.
- [ ] **M8.6** `gosec` + `govulncheck` in CI; dependency audit.
- [ ] **M8.7** Documentation: README, admin guide, operator guide for Traefik + oauth2-proxy + common IdPs.

## Milestone 9 — Local Dev Stack (k3d)

- [ ] **M9.1** `k3d` cluster config with Traefik v2 preinstalled.
- [ ] **M9.2** Dex deployment with static client `queryexplorer` and mock connector; seed three users (admin+PII, editor+PII, viewer-no-PII) with group claims.
- [ ] **M9.3** oauth2-proxy deployment configured against Dex; `--pass-user-headers`, `--set-xauthrequest`.
- [ ] **M9.4** Traefik `Middleware` (`ForwardAuth`) + header-strip middleware for inbound `X-User-*`.
- [ ] **M9.5** Traefik `Middleware` for injecting `X-Auth-Proxy-Secret` from a Kubernetes Secret.
- [ ] **M9.6** IngressRoute for `queryexplorer.local.test` wiring all middlewares in order.
- [ ] **M9.7** Target Postgres deployment with seed data including a PII column (personnummer) for masking tests.
- [ ] **M9.8** `make dev-up` / `make dev-down` scripts; `/etc/hosts` note in README.
- [ ] **M9.9** End-to-end smoke test script: login as each of the three Dex users and hit `/api/me`, run a masked query, verify masking differs per user.

## Milestone 10 — Optional / v1.1

- [ ] **M10.1** Audit archival job: rotate old runs to S3 as compressed JSONL.
- [ ] **M10.2** Query favorites / pinning (keyed by `user_sub`; still no users table — just a `favorites` table with sub as FK-free column).
- [ ] **M10.3** Query folders / tags.
- [ ] **M10.4** Result diffing between two runs of the same query.
- [ ] **M10.5** Additional drivers: MySQL, SQL Server, ClickHouse.
- [ ] **M10.6** Per-query "require group" setting (not just masking — block execution entirely for groups).

---

## Acceptance Criteria Summary

The v1 release is done when:

1. Traefik + oauth2-proxy + Dex in k3d can authenticate three distinct test users, and QueryExplorer reads their identity from headers with proxy-secret verification.
2. An admin can create a PostgreSQL connection with an encrypted DSN.
3. An editor can create a saved query with typed parameters and declared column masks.
4. Three users with different group memberships running the same query produce three different visible result sets and three audit rows correctly recording `masked_columns`.
5. A viewer can export a saved query to CSV and XLSX within their row cap; each export is recorded as its own run with `export_format` set.
6. An editor can run ad-hoc SQL against a read-only connection; write statements are rejected.
7. Cancelling a browser tab during a long-running query marks the run as `cancelled` within the connection timeout window.
8. An admin can filter the audit view, drill into any run, and see full SQL, parameters, groups at execution, and masked columns.
9. Spoofed `X-User-*` headers without the proxy secret receive 401.
