# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository scope

The working directory `/home/hediad/projects/waf` contains a vendored upstream: `ModSecurity/` — a clone of `owasp-modsecurity/ModSecurity` (libmodsecurity v3). The runtime WAF currently uses the Docker image `owasp/modsecurity-crs:nginx-alpine`, while `ModSecurity/` is kept in-tree for future inspection, patching, and custom builds. New WAF/dashboard code should usually live as **siblings** of `ModSecurity/`, but edits inside `ModSecurity/` are allowed when the task explicitly requires changing libmodsecurity.

## Architecture and stack

The full target architecture, stack choices, repo layout and roadmap are documented in **[ARCHITECTURE.md](./ARCHITECTURE.md)** — read it before adding new components or making structural decisions. Short summary:

- **Data plane:** Nginx + ModSecurity-nginx connector + OWASP CRS (libmodsecurity is consumed via the connector — do not write a custom HTTP proxy).
- **Ingester:** Go 1.23+ — tails JSON audit log, enriches with GeoIP/ASN, writes to ClickHouse, publishes to Redis pub/sub.
- **Storage:** ClickHouse (events), PostgreSQL 16 (rules/users/config), Redis 7 (pub/sub + cache).
- **API:** Go + chi + sqlc — REST + Server-Sent Events for live stream.
- **Dashboard:** Next.js 15 + TypeScript + shadcn/ui + Tremor + Tailwind, react-leaflet for the GeoIP map.
- **Dev deploy:** `docker-compose.yml` at repo root spins up the entire stack.

If the stack changes, update `ARCHITECTURE.md` first, then this section.

## What `ModSecurity/` is (and isn't)

- It is **libmodsecurity** — a C++17 library that parses SecRules (the ModSecurity rules language) and applies them to HTTP transactions handed in via a connector. It exposes both a C++ API (`ModSecurity::ModSecurity`, `RulesSet`, `Transaction`) and a C API (`msc_init`, `msc_create_rules_set`, `msc_new_transaction`, `msc_process_*`).
- It is **not** a standalone WAF and is **not** a webserver module. It has no Apache/Nginx/IIS code in this repo. To get HTTP traffic into the engine, code must either (a) consume an existing connector (e.g. `ModSecurity-nginx`), or (b) drive the C/C++ API directly from a custom proxy/server. See `examples/simple_example_using_c/test.c` and `examples/multithread/multithread.cc` for the minimum integration path.
- The transaction lifecycle the integrator must drive, in order, is: `processConnection` → `processURI` → `processRequestHeaders` → `processRequestBody` → `processResponseHeaders` → `processResponseBody` → `processLogging`. After each phase, check `transaction->intervention()` to learn whether a rule wants to block/redirect.

## Building libmodsecurity

```shell
cd ModSecurity
git submodule update --init --recursive   # required: libinjection, mbedtls, test cases
./build.sh                                 # regenerates headers.mk + autotools files
./configure                                # use --enable-assertions=yes when debugging
make -j$(nproc)
sudo make install                          # installs to /usr/local/modsecurity by default
```

`build.sh` regenerates `src/headers.mk` from the on-disk header layout and runs `libtoolize` + `autoreconf`. Re-run it any time headers are added/removed/renamed under `src/`.

For debug builds: `export CFLAGS="-g -O0" CXXFLAGS="-g -O0"` before `./configure --enable-assertions=yes`.

## Running tests

The test target is wired through automake. Unit tests live under `test/unit/` (driven by `test/unit_tests`); regression tests are JSON-driven scenarios under `test/test-cases/regression/` and `test/test-cases/secrules-language-tests/` (driven by `test/regression_tests`). Both binaries are built by `make`.

```shell
cd ModSecurity
make check                                              # runs everything in test/test-suite.in
./test/regression_tests test/test-cases/regression/operator-rx.json     # one whole JSON file
./test/regression_tests test/test-cases/regression/operator-rx.json:3   # one case (1-indexed)
./test/regression_tests countall test/test-cases/regression/foo.json    # how many cases in file
./test/unit_tests test/test-cases/secrules-language-tests/operators/rx.json
```

A skipped test result with message `json is not enabled` means the build was configured without YAJL — install YAJL and reconfigure.

## Static analysis and style

- `make cppcheck` — runs cppcheck with `test/cppcheck_suppressions.txt` and exits non-zero on findings (skips the generated parser/scanner).
- `make check-coding-style` — runs `cpplint.py` and writes `coding-style.txt`. The project follows the Chromium/Blink C++ style.
- Both the parser (`src/parser/seclang-parser.{cc,hh}`) and scanner (`src/parser/seclang-scanner.cc`) are **generated** from Yacc/Flex sources — do not hand-edit. The `parser` target in the top-level Makefile post-processes the generated header to fix a move-semantics issue.

## Benchmarking

```shell
cd ModSecurity/test/benchmark
./download-owasp-v3-rules.sh    # or download-owasp-v4-rules.sh, for realistic rule load
./benchmark 10000               # default 1,000,000 transactions; pass a number to override
```

The benchmark builds in-tree and uses `basic_rules.conf` next to it; reset that file when switching rulesets.

## Source layout (libmodsecurity)

The interesting subsystems all live under `ModSecurity/src/` with public headers mirrored in `ModSecurity/headers/modsecurity/`:

- `modsecurity.cc` / `transaction.cc` / `rules_set.cc` — the top-level engine, the per-request transaction, and the rule container. These are the entry points an integrator touches.
- `parser/` — Flex+Yacc grammar for the SecRules language. Generated files (`seclang-parser.*`, `seclang-scanner.cc`) are checked in.
- `actions/` — every disruptive (`actions/disruptive/`), control (`actions/ctl/`), data (`actions/data/`), and transformation (`actions/transformations/`) action a rule can declare. To add a new action, drop a `.h/.cc` pair here and re-run `./build.sh` so `headers.mk` picks it up.
- `operators/` — rule operators (`@rx`, `@detectSQLi`, `@detectXSS`, `@ipMatch`, `@pmFromFile`, …). Same add-a-file workflow as actions.
- `variables/` — implementations of every collection/variable referenced from rules (`ARGS`, `REQUEST_HEADERS`, `TX`, `GEO`, …).
- `request_body_processor/` — JSON, XML, multipart, urlencoded body parsers wired to `REQBODY_PROCESSOR`.
- `audit_log/` and `debug_log/` — the audit log (serial/concurrent/HTTPS writers under `audit_log/writer/`) and the debug log subsystems. **A future dashboard most likely ingests audit log output from here** — adding a new writer (e.g. JSON-over-socket, syslog/NDJSON) is the natural integration point and is parallel to the existing serial/concurrent writers.
- `collection/` — persistent collection backends (in-memory, LMDB) backing `IP`, `SESSION`, `USER`, `RESOURCE`, `TX`, `GLOBAL`.
- `engine/` — the embedded Lua engine for `SecRuleScript`.
- `utils/` — shared helpers (regex, geo, msc_string, etc.).

## Required submodules

`others/libinjection` (SQLi/XSS detection) and `others/mbedtls` (crypto) are mandatory submodules; `./configure` will refuse to proceed without them. `bindings/python` and `test/test-cases/secrules-language-tests` are also submodules. Always run `git submodule update --init --recursive` after a fresh clone or branch switch.

## Notes on dual-licensing realities

ModSecurity is Apache-2.0 licensed and the upstream Trustwave sponsorship ended 2024-07-01 (per `ModSecurity/README.md`); the project continues under OWASP. Keep third-party notices with the repo, and if files inside `ModSecurity/` are modified for redistribution, mark those files as changed per Apache-2.0.
