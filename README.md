# WAF Dashboard

`waf` is a Docker-first web application firewall stack:

- `nginx` runs the WAF data plane using the `owasp/modsecurity-crs:nginx-alpine` image.
- `ModSecurity/` stores a vendored upstream copy of `libmodsecurity` for future inspection, patching, and custom builds.
- `ingester` tails ModSecurity audit logs and nginx access logs, enriches events, and writes them to ClickHouse plus Redis.
- `api` exposes event queries and a live Server-Sent Events stream.
- `dashboard` renders the monitoring UI.

## Runtime flow

1. Traffic hits nginx with ModSecurity and OWASP CRS enabled.
2. Allowed requests are proxied to the origin service.
3. Blocked requests produce ModSecurity JSON concurrent audit log files.
4. `ingester` converts audit and access logs into normalized events.
5. `api` reads from ClickHouse and streams live updates from Redis.
6. `dashboard` renders overview and event pages from the API.

## Repository layout

- `ModSecurity/`: vendored upstream `owasp-modsecurity/ModSecurity` sources kept in-tree for future modification.
- `nginx/`: ModSecurity overrides and custom SecRule files.
- `ingester/`: Go log ingester.
- `api/`: Go HTTP API and SSE stream.
- `dashboard/`: Next.js frontend.
- `deploy/`: database bootstrap SQL and deployment assets.

## Git notes

This repository is intended to be a single root git repository.

- Historical nested git metadata for vendored components is stored under `.git-archives/` and ignored by git.
- `ModSecurity/` is tracked as normal source code so future local modifications can be committed together with the rest of the stack.

## Licenses

- `ModSecurity/` is Apache-2.0 licensed. See `ModSecurity/LICENSE`.
- Third-party attribution notes are summarized in `THIRD_PARTY_NOTICES.md`.
