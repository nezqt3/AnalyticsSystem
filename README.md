# AnalyticsSystem

Система аналитики поведения пользователей с встраиваемым JS-трекером, Go backend и React dashboard.

## Что теперь есть
- Встраивание остаётся через `script src`: достаточно подключить `tracker.js` и передать `data-site` + `data-endpoint`.
- Трекер собирает `pageview`, `click`, `scroll`, `form_submit` и корректно отслеживает SPA-навигацию.
- Backend разбит на слои: `cmd`, `internal/app`, `internal/httpapi`, `internal/store`, `internal/model`, `internal/tracker`.
- Dashboard показывает список страниц, realtime, heatmap, top click targets, глубину скролла, события и источники трафика.
- Для проекта настроены `Makefile`, `ESLint`, `Prettier`, `pre-commit`, CI/CD и автоматические тесты для frontend и backend.

## Структура
- `backend/cmd/server` — точка входа backend.
- `backend/internal/app` — инициализация приложения и фоновые maintenance-задачи.
- `backend/internal/httpapi` — HTTP API и WebSocket.
- `backend/internal/store/sqlstore` — SQL-слой, агрегации, retention, выборки аналитики.
- `backend/internal/tracker` — встроенный JS-трекер.
- `frontend/src/components` — UI-блоки dashboard.
- `frontend/src/hooks` — загрузка данных и realtime.
- `frontend/src/lib` — утилиты форматирования и разбора метаданных.
- `frontend/src/types` — типы API.
- `frontend/src/**/*.test.ts(x)` — Jest-тесты для UI и утилит.
- `backend/internal/**/*_test.go` — тесты backend.
- `.github/workflows` — CI/CD пайплайны.
- `.pre-commit-config.yaml` — локальные pre-commit проверки.
- `Makefile` — основные команды разработки.

## Быстрый старт

### Backend
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

### Tracker
Подключение на сайт:

```html
<script
  src="http://localhost:8080/tracker.js"
  data-site="1"
  data-endpoint="http://localhost:8080/collect"
></script>
```

### Frontend
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

Покрываются:
- рендеринг ключевых UI-блоков
- поведение dashboard при загрузке данных
- разбор и отображение метаданных событий

### Backend
Используются стандартные Go tests c SQLite test database.

Команды:

```bash
cd backend
go test ./...
go test ./... -coverprofile=coverage.out
go vet ./...
```

Покрываются:
- парсинг входящих событий
- SQL store и агрегации по страницам
- HTTP handlers для ingest и аналитики

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

## Frontend scripts
```bash
cd frontend
npm run lint
npm run lint:fix
npm run typecheck
npm run format
npm run format:check
npm run test
npm run test:coverage
npm run build
```

## API
- `GET /api/pages` — список страниц с просмотрами, кликами и посетителями.
- `GET /api/page-analytics` — детальная аналитика по одной странице.
- `GET /api/realtime` — realtime за последние 5 минут.
- `GET /api/heatmap` — тепловая карта кликов.
- `GET /api/traffic-sources` — источники трафика.
- `GET /api/events` — последние события.
- `POST /collect` — приём батчей событий от трекера.

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
