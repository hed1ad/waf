# WAF Dashboard

This project is built around the `ModSecurity` engine.

It takes `ModSecurity` as the core WAF and rule engine, then adds a clean, polished dashboard on top for monitoring events, reviewing attacks, and managing the system more comfortably.

The current stack looks like this:

- `nginx` + `ModSecurity` + OWASP CRS inspect and filter HTTP traffic
- `ingester` reads audit logs and access logs
- `api` serves events, statistics, and a live stream
- `dashboard` presents charts, activity feeds, and attack summaries

## Project Idea

The goal is not to build a WAF engine from scratch. The goal is to take a mature engine like `ModSecurity` and build a usable product layer around it:

- a clear interface
- centralized event visibility
- aggregated security analytics
- a foundation for rule and policy management

## Repository Structure

- `ModSecurity/` — vendored `libmodsecurity` upstream sources that can be modified locally
- `nginx/` — WAF-layer configuration and custom rules
- `ingester/` — event collection and normalization
- `api/` — backend API and live stream
- `dashboard/` — frontend dashboard
- `deploy/` — SQL bootstrap and deployment support files

## What Exists Today

- a Docker-first environment for running the full stack
- `ModSecurity` as the WAF core
- audit and access event collection
- ClickHouse-based event storage
- an API for querying and aggregating data
- a dashboard for observing traffic and attacks

## Planned Next

The project is expected to grow toward more enterprise-oriented capabilities, including:

- SSO
- integration with corporate identity providers
- more complete user and role management
- access control and administrative features
- more advanced rule and policy management

## Git and Licensing

- `ModSecurity/` is kept in the repository as normal source code so it can be modified together with the rest of the project
- historical nested git metadata is preserved locally under `.git-archives/`
- `ModSecurity` is distributed under Apache-2.0, see `ModSecurity/LICENSE`
- additional third-party notices are documented in `THIRD_PARTY_NOTICES.md`
