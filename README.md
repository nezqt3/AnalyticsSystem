# AnalyticsSystem

Система аналитики поведения пользователей с встраиваемым JS-трекером, защищённым admin dashboard, Go backend и React frontend.

## Что теперь умеет система
- Встраивание остаётся через `script src`: достаточно подключить `tracker.js` и передать `data-site` + `data-endpoint`.
- Трекер собирает `pageview`, `click`, `scroll`, `form_submit` и отслеживает SPA-навигацию.
- Dashboard показывает:
  - сводные KPI по просмотрам, кликам и посетителям,
  - график динамики входов по времени,
  - график входов по страницам,
  - последние входы пользователей: когда и на какую страницу зашли,
  - realtime-активность,
  - heatmap,
  - top click targets,
  - глубину скролла,
  - таблицу сырых событий.
- Все API dashboard защищены авторизацией одного admin-аккаунта через `env`.
- Для проекта настроены `Makefile`, `ESLint`, `Prettier`, `pre-commit`, CI/CD и тесты для frontend и backend.

## Admin auth
Используется один admin-аккаунт, данные задаются через переменные окружения:
- `ADMIN_EMAIL`
- `ADMIN_PASSWORD`
- `SESSION_SECRET`

Пример есть в `/Users/denisalekseenko/Documents/Сайты/AnalyticsSystem/.env.example`.

## Структура
- `backend/cmd/server` — точка входа backend.
- `backend/internal/app` — инициализация приложения и maintenance-задачи.
- `backend/internal/auth` — логика admin-сессий.
- `backend/internal/httpapi` — HTTP API, auth middleware, WebSocket.
- `backend/internal/store/sqlstore` — SQL-слой, агрегации и выборки аналитики.
- `backend/internal/tracker` — встроенный JS-трекер.
- `frontend/src/components` — UI-компоненты dashboard.
- `frontend/src/hooks` — auth, realtime и загрузка аналитики.
- `frontend/src/lib` — утилиты форматирования и разбора метаданных.
- `frontend/src/types` — типы API.
- `frontend/src/**/*.test.ts(x)` — Jest-тесты frontend.
- `backend/internal/**/*_test.go` — тесты backend.
- `.github/workflows` — CI/CD пайплайны.
- `.pre-commit-config.yaml` — локальные проверки перед коммитом.
- `Makefile` — основные команды разработки.

## Быстрый старт

### 1. Backend
```bash
cd backend
go run ./cmd/server
```

Переменные окружения:
- `PORT` — порт backend, по умолчанию `8080`
- `DATABASE_URL` — подключение к Postgres
- `SQLITE_PATH` — путь к SQLite, по умолчанию `./data/analytics.db`
- `RAW_RETENTION_DAYS` — хранение сырых событий, по умолчанию `30`
- `AGG_RETENTION_MONTHS` — хранение агрегатов, по умолчанию `12`
- `HEATMAP_BUCKET_PCT` — размер ячейки heatmap, по умолчанию `5`
- `EVENTS_LIMIT` — дефолтный лимит событий, по умолчанию `200`
- `ADMIN_EMAIL` — email администратора
- `ADMIN_PASSWORD` — пароль администратора
- `SESSION_SECRET` — секрет для подписи admin-сессии

### 2. Tracker
Подключение на сайт:

```html
<script
  src="http://localhost:8080/tracker.js"
  data-site="1"
  data-endpoint="http://localhost:8080/collect"
></script>
```

### 3. Frontend
```bash
cd frontend
npm install
npm run dev
```

## Makefile
Основные команды:

```bash
make install
make install-pre-commit
make dev-backend
make dev-frontend
make lint
make typecheck
make test
make test-coverage
make build
make ci
```

## Тесты
### Frontend
Используется `Jest` + `Testing Library`.

Команды:

```bash
cd frontend
npm run test
npm run test:watch
npm run test:coverage
```

### Backend
Используются Go unit/integration tests с SQLite test database.

Команды:

```bash
cd backend
go test ./...
go test ./... -coverprofile=coverage.out
go vet ./...
```

## API для dashboard
- `POST /api/auth/login` — вход администратора.
- `POST /api/auth/logout` — выход.
- `GET /api/auth/me` — текущий admin.
- `GET /api/overview` — сводные метрики.
- `GET /api/timeline` — входы по времени.
- `GET /api/visits` — последние входы пользователей по страницам.
- `GET /api/pages` — список страниц с просмотрами, кликами и посетителями.
- `GET /api/page-analytics` — детальная аналитика по одной странице.
- `GET /api/realtime` — realtime за последние 5 минут.
- `GET /api/heatmap` — тепловая карта кликов.
- `GET /api/traffic-sources` — источники трафика.
- `GET /api/events` — последние события.
- `POST /collect` — приём батчей событий от трекера.

## Pre-commit
Установка хуков:

```bash
pre-commit install
```

Или через Makefile:

```bash
make install-pre-commit
```

Что проверяется перед коммитом:
- базовые проверки структуры файлов
- `gofmt`
- `go test ./...`
- `go vet ./...`
- `npm run lint`
- `npm run typecheck`
- `npm run format:check`
- `npm run test -- --runInBand`

## CI/CD
### CI
Workflow `/Users/denisalekseenko/Documents/Сайты/AnalyticsSystem/.github/workflows/ci.yml` запускает:
- backend format check
- backend `go vet`
- backend tests
- backend build
- frontend prettier check
- frontend eslint
- frontend typecheck
- frontend jest
- frontend build

### CD
Workflow `/Users/denisalekseenko/Documents/Сайты/AnalyticsSystem/.github/workflows/cd.yml` на `main`:
- собирает Docker image backend
- собирает Docker image frontend
- пушит их в `ghcr.io`

## Docker
```bash
docker compose up --build
```

Сервисы:
- backend: `http://localhost:8080`
- frontend: `http://localhost:5173`
- postgres: `localhost:5432`
