# AnalyticsSystem (MVP)

Минимальный старт для аналитической платформы (Google Analytics lite).

## Что уже есть
- Go backend: сбор событий, realtime, heatmaps, источники трафика.
- JS трекер: pageview + клики.
- TS frontend: простая панель.

## Быстрый старт

### 1) База данных
Локально по умолчанию используется SQLite (`./data/analytics.db`), схема применится автоматически.

Если нужен Postgres локально:
- выставь `DATABASE_URL`,
- схема применится автоматически из `backend/db/schema.pg.sql`.

### 2) Backend (локально)

```bash
cd backend
go run .
```

Опционально:
- `DATABASE_URL` для Postgres
- `SQLITE_PATH` для SQLite (по умолчанию `./data/analytics.db`)

### 3) Tracker
Подключи на сайт:

```html
<script
  src="http://localhost:8080/tracker.js"
  data-site="1"
  data-endpoint="http://localhost:8080/collect"
></script>
```

### 4) Frontend

```bash
cd frontend
npm install
npm run dev
```

Переменные окружения фронта:
- `VITE_API_BASE` (по умолчанию `http://localhost:8080`)
- `VITE_SITE_ID` (по умолчанию `1`)

## Docker
Запуск с Postgres:

```bash
docker compose up --build
```

Сервисы:
- backend: `http://localhost:8080`
- frontend: `http://localhost:5173`
- postgres: `localhost:5432`

## Следующие шаги (рекомендуемые)
- Авторизация и мультисайтовость.
- Нормализация источников (utm, referrer, direct).
- Очистка и агрегация событий (retention).
- Хранилище для heatmap по страницам и размерам экрана.
- Очередь и батчинг для событий.

## Авто-агрегации и retention
Backend сам:
- создаёт месячные партиции `events_YYYY_MM` (на 3 месяца вперёд),
- пересчитывает дневные агрегаты за последние 2 дня,
- удаляет старые сырые события и старые агрегаты.

Переменные окружения:
- `RAW_RETENTION_DAYS` (по умолчанию 30)
- `AGG_RETENTION_MONTHS` (по умолчанию 12)

## Нормализация источников и фильтры
Трекер отправляет `utm_source`, `utm_medium`, `utm_campaign`, `entry_url`.
Backend нормализует источники:
- приоритет `utm_source`,
- затем домен реферера,
- иначе `direct`.

Фильтры в API `/api/traffic-sources`:
- `path` (опционально),
- `from` и `to` (YYYY-MM-DD).

## Realtime WebSocket и батчинг
- Endpoint: `ws://<host>/ws/realtime?site_id=1`
- При недоступности WS фронт делает polling `/api/realtime`.
- Трекер отправляет события пачками (до 20 или раз в 5 сек).

## Heatmap (улучшенная)
- API `/api/heatmap` теперь возвращает проценты по экрану (`x_pct`, `y_pct`).
- Параметр `bucket` (1-25) задаёт размер ячейки в процентах.
- Дефолт можно задать через `HEATMAP_BUCKET_PCT` (по умолчанию 5).
