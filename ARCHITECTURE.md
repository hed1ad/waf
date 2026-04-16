# Архитектура WAF + Dashboard

Документ описывает целевой стек и архитектуру проекта. Изменения в стеке должны отражаться здесь и в CLAUDE.md.

## Высокоуровневая схема

```
┌────────┐  HTTP   ┌──────────────────────┐  proxy   ┌─────────┐
│ Client │ ──────▶ │ Nginx + ModSecurity  │ ───────▶│ Origin  │
└────────┘         │  + OWASP CRS rules   │          └─────────┘
                   └──────────┬───────────┘
                              │ JSON audit log (Unix socket / file tail)
                              ▼
                   ┌──────────────────────┐
                   │ Ingester (Go)        │  парсинг, нормализация,
                   │  tail + enrich       │  GeoIP, ASN, fingerprint
                   └──────────┬───────────┘
                              ▼
        ┌─────────────────────┴─────────────────────┐
        ▼                     ▼                     ▼
  ┌──────────┐         ┌────────────┐        ┌─────────┐
  │ClickHouse│         │ PostgreSQL │        │  Redis  │
  │  events  │         │ rules/users│        │ pubsub  │
  └────┬─────┘         └─────┬──────┘        └────┬────┘
       └──────────┬──────────┴────────────────────┘
                  ▼
         ┌──────────────────┐
         │   API (Go)       │   REST + SSE
         │   chi + sqlc     │   /events /rules /stats /stream
         └────────┬─────────┘
                  ▼
         ┌──────────────────┐
         │  Next.js 15      │   live-чарты, фильтры, правила,
         │  shadcn/ui+Tremor│   GeoIP-карта, dark mode
         └──────────────────┘
```

## Слои и ответственность

### 1. Data plane — Nginx + ModSecurity-nginx + OWASP CRS
- Терминирует HTTP/HTTPS, проксирует на origin.
- Runtime сейчас идёт через Docker-образ `owasp/modsecurity-crs:nginx-alpine`.
- `libmodsecurity` также хранится vendored в `ModSecurity/` для будущих локальных модификаций и кастомных сборок.
- Правила: OWASP Core Rule Set v4 + кастомные правила из Postgres.
- Аудит-лог формат: **JSON Concurrent** (`SecAuditLogFormat JSON`, `SecAuditLogType Concurrent`) — каждое событие отдельным файлом или поток в Unix socket.

### 2. Ingester — Go 1.23+
- Один статический бинарь, без runtime-зависимостей.
- `fsnotify` для tail аудит-лога (или syslog/UDP receiver).
- Парсинг JSON → нормализация → batch insert в ClickHouse.
- Обогащение событий:
  - **GeoIP** — MaxMind GeoLite2 City + ASN.
  - **TLS fingerprint** — JA3/JA4 (если доступен из nginx).
  - **Dedup** — схлопывание идентичных атак в окне.
- Публикация в Redis pub/sub для live-стрима в дашборд.

### 3. Хранилища

| Что хранится | СУБД | Почему |
|---|---|---|
| События (атаки, hits, blocks) | **ClickHouse** | OLAP, быстрые агрегации по миллионам строк, time-series |
| Конфиг: правила, юзеры, списки, настройки | **PostgreSQL 16** | Реляционные данные, транзакции, миграции |
| Live-стрим, кэш агрегатов, сессии | **Redis 7** | Pub/sub + low-latency cache |

### 4. API — Go + chi + sqlc
- HTTP роутер: `chi` — минимализм, middleware-friendly.
- SQL: **sqlc** — типобезопасный код из SQL-запросов, без ORM.
- Эндпоинты:
  - `GET  /api/events` — поиск/фильтр/пагинация
  - `GET  /api/events/:id` — детали транзакции
  - `GET  /api/stats/*` — агрегаты для чартов
  - `CRUD /api/rules` — управление правилами
  - `CRUD /api/lists` — белые/чёрные списки IP
  - `GET  /api/stream` — live feed через Server-Sent Events (SSE)
- Auth: JWT + сессия в Redis. Опционально OIDC.
- Перезагрузка nginx после CRUD правил: рендер `.conf` → `nginx -t` → `nginx -s reload`.

### 5. Dashboard — Next.js 15 + TypeScript
- **App Router + RSC** — SEO, быстрые первые рендеры.
- **shadcn/ui** — компоненты на Radix + Tailwind.
- **Tremor** / **Recharts** — графики (timeseries, bar, donut, heatmap).
- **react-leaflet** + GeoIP — карта источников атак.
- **TanStack Query** — кэш данных, infinite scroll по событиям.
- **Tailwind CSS** + dark mode.
- SSE-клиент для `/api/stream`.

Основные экраны:
- Overview — KPI карточки, timeseries, top rules, top countries.
- Events — таблица + детали транзакции (request/response/matched rules).
- Rules — CRUD кастомных правил, включение/отключение CRS-категорий.
- Lists — IP allow/deny.
- Settings — пользователи, ключи API, retention.

### 6. Deploy
- **Dev:** `docker compose up` — nginx, modsecurity, clickhouse, postgres, redis, ingester, api, frontend в одной сети.
- **Prod:** Helm chart для Kubernetes (позже).

## Целевая структура репозитория

```
waf/
├── ModSecurity/          # vendored upstream libmodsecurity (можно модифицировать локально)
├── README.md
├── THIRD_PARTY_NOTICES.md
├── ARCHITECTURE.md       # этот файл
├── CLAUDE.md             # инструкции для Claude
├── docker-compose.yml    # dev окружение целиком
├── nginx/                # конфиги nginx + modsec + CRS
│   ├── nginx.conf
│   ├── modsecurity.conf
│   └── rules/
├── ingester/             # Go: tail аудит-лога → ClickHouse + Redis
│   ├── cmd/ingester/
│   ├── internal/parser/
│   ├── internal/enrich/
│   └── internal/sink/
├── api/                  # Go: REST + SSE
│   ├── cmd/api/
│   ├── internal/handlers/
│   ├── internal/store/   # sqlc-сгенерированный код
│   └── migrations/
├── dashboard/            # Next.js 15 frontend
│   ├── app/
│   ├── components/
│   └── lib/
└── deploy/               # Helm chart, prod-конфиги (позже)
```

## План разработки (этапы)

1. **Boot** — `docker-compose` с nginx + modsecurity + CRS, проксирующим на тестовый origin. Аудит-лог пишется в файл.
2. **Ingester** — Go-сервис, читает аудит-лог, пишет в ClickHouse и публикует в Redis.
3. **API skeleton** — Go-сервис, чтение событий из ClickHouse, базовые stats эндпоинты.
4. **Dashboard MVP** — Next.js, Overview-страница (3 чарта + live feed через WS).
5. **Управление правилами** — CRUD в API, рендер `.conf`, reload nginx.
6. **GeoIP-обогащение + карта** в дашборде.
7. **Auth** (JWT + OIDC), retention, alerts.
8. **Helm chart** для production.

## Альтернативный lite-стек (если понадобится MVP за час)

| Слой | Production | Lite MVP |
|------|-----------|----------|
| WAF | Nginx + ModSec | то же |
| Ingester | Go | Python (watchdog) |
| Events DB | ClickHouse | SQLite (FTS5) |
| Config DB | PostgreSQL | SQLite |
| Cache/PubSub | Redis | in-process |
| API | Go + chi + sqlc | FastAPI + SQLAlchemy |
| Frontend | Next.js 15 | Vite + React |

Lite — для прототипа, production — для масштабирования до миллионов событий/сутки.
