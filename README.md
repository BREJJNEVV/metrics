# Iter 17
Добавил батчевый запрос в бд в UpdatesJSON, но памяти стало аллоцирваться гораздо больше, пришлось удалить(показывало около 1032 МБ).

Тестировал с помощью HEY на реальном снимке метрик test_data.json

hey -n 10000 -c 50 \
  -m POST \
  -T "application/json" \
  -D test_data.json \
  http://localhost:8080/updates


ivanshnyakin@MacBook-Air-Ivan metrics % go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
File: main
Type: inuse_space
Time: 2026-11-17 18:00:53 MSK
Showing nodes accounting for 1018.33kB, 32.92% of 3092.93kB total
Dropped 2 nodes (cum <= 15.46kB)
      flat  flat%   sum%        cum   cum%
 -525.43kB 16.99% 16.99%  -525.43kB 16.99%  github.com/jackc/pgx/v5/internal/stmtcache.NewLRUCache (inline)
  521.05kB 16.85%  0.14%   521.05kB 16.85%  encoding/xml.map.init.0
 -515.19kB 16.66% 16.80%  -515.19kB 16.66%  unicode.map.init.2
     514kB 16.62%  0.18%      514kB 16.62%  bytes.growSlice
 -513.31kB 16.60% 16.78% -1038.74kB 33.58%  github.com/jackc/pgx/v5/pgxpool.NewWithConfig.func3
     513kB 16.59%  0.19%      513kB 16.59%  runtime.mallocgc
  512.16kB 16.56% 16.37%   512.16kB 16.56%  net/http.(*Request).WithContext (inline)
  512.05kB 16.56% 32.92%   512.05kB 16.56%  github.com/golang-migrate/migrate/v4.(*Migrate).lock.func2
         0     0% 32.92%      514kB 16.62%  bytes.(*Buffer).ReadFrom
         0     0% 32.92%      514kB 16.62%  bytes.(*Buffer).grow
         0     0% 32.92%   521.05kB 16.85%  encoding/xml.init
         0     0% 32.92%      514kB 16.62%  github.com/BREJJNEVV/metrics/internal/handler.(*MetricService).UpdatesJSON
         0     0% 32.92%      514kB 16.62%  github.com/BREJJNEVV/metrics/internal/handler.GzipCompress.func1
         0     0% 32.92%      514kB 16.62%  github.com/BREJJNEVV/metrics/internal/handler.GzipDecompress.func1
         0     0% 32.92%      514kB 16.62%  github.com/go-chi/chi.(*ChainHandler).ServeHTTP
         0     0% 32.92%  1026.16kB 33.18%  github.com/go-chi/chi.(*Mux).ServeHTTP
         0     0% 32.92%      514kB 16.62%  github.com/go-chi/chi.(*Mux).routeHTTP
         0     0% 32.92%  -525.43kB 16.99%  github.com/jackc/pgx/v5.ConnectConfig
         0     0% 32.92%  -525.43kB 16.99%  github.com/jackc/pgx/v5.connect
         0     0% 32.92% -1038.74kB 33.58%  github.com/jackc/puddle/v2.(*Pool[go.shape.*uint8]).initResourceValue.func1
         0     0% 32.92%      514kB 16.62%  main.main.WithLogging.func4.1
         0     0% 32.92%     -513kB 16.59%  net/http.(*conn).readRequest
         0     0% 32.92%  1027.16kB 33.21%  net/http.(*conn).serve
         0     0% 32.92%      514kB 16.62%  net/http.HandlerFunc.ServeHTTP
         0     0% 32.92%  1026.16kB 33.18%  net/http.serverHandler.ServeHTTP
         0     0% 32.92%      513kB 16.59%  runtime.allocm
         0     0% 32.92%     5.86kB  0.19%  runtime.doInit (inline)
         0     0% 32.92%     5.86kB  0.19%  runtime.doInit1
         0     0% 32.92%     5.86kB  0.19%  runtime.main
         0     0% 32.92%      513kB 16.59%  runtime.mcall
         0     0% 32.92%      513kB 16.59%  runtime.newm
         0     0% 32.92%      513kB 16.59%  runtime.newobject
         0     0% 32.92%      513kB 16.59%  runtime.park_m
         0     0% 32.92%      513kB 16.59%  runtime.resetspinning
         0     0% 32.92%      513kB 16.59%  runtime.schedule
         0     0% 32.92%      513kB 16.59%  runtime.startm
         0     0% 32.92%      513kB 16.59%  runtime.wakep
         0     0% 32.92%  -515.19kB 16.66%  unicode.init
ivanshnyakin@MacBook-Air-Ivan metrics % 















# Metrics Collection & Alerting Service

Сервис сбора, хранения и алертинга рантайм-метрик — учебный проект курса «Go-разработчик» Яндекс.Практикума.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## 📖 О проекте

**Metrics Service** — клиент-серверная система для сбора рантайм-метрик приложения и их последующей обработки.

- **Агент** периодически опрашивает пакет `runtime` и системные метрики через `gopsutil`, а затем отправляет их на сервер батчами.
- **Сервер** принимает метрики по HTTP, хранит их в памяти, файле или PostgreSQL, отдаёт по запросу и отображает в HTML.

## ✨ Возможности

### Агент

- 📈 Сбор 28+ метрик из `runtime` (`Alloc`, `Sys`, `HeapAlloc`, `GC*` и другие).
- 🖥️ Сбор системных метрик через `gopsutil` (`TotalMemory`, `FreeMemory`, `CPUutilizationN`).
- 🔁 Отправка батчами с интервалом `reportInterval`.
- 🗜️ Gzip-сжатие тела запроса.
- 🔐 Подпись HMAC-SHA256 в формате Base64.
- ⚙️ Worker pool с ограничением параллельных запросов (`-l`).
- 🛡️ Retry с экспоненциальной задержкой (1, 3 и 5 секунд) для временных сетевых ошибок.

### Сервер

- 🚀 HTTP API на `chi` с цепочкой middleware.
- 💾 Три бэкенда хранения: память, файл и PostgreSQL.
- 📦 Автоматические миграции через `golang-migrate` и `embed.FS`.
- 🔄 Батчевая обработка метрик (`POST /updates`) с транзакциями.
- 🗜️ Gzip: приём сжатых тел запросов и отправка сжатых ответов.
- 🔐 Проверка HMAC-SHA256 и подпись ответов.
- 📊 HTML-страница со списком метрик.
- ❤️ Health check: `GET /ping`.
- 🧾 Структурированное логирование через `zap`.
- 🔁 Retry при ошибках PostgreSQL классов `08` и `40`.

## 🏗️ Архитектура

```text
┌─────────────┐         HTTP (JSON + gzip + HMAC)        ┌────────────────┐
│             │ ───────────────────────────────────────▶ │                │
│   Agent     │                                          │    Server      │
│  (worker    │ ◀─────────────────────────────────────── │  (chi router)  │
│   pool)     │              response + hash             │                │
└─────────────┘                                          └───────┬────────┘
                                                                 │
                                                                 ▼
                                              ┌──────────────────────────────┐
                                              │  Repository (interface)      │
                                              ├──────────────────────────────┤
                                              │  memory  │  file  │  postgres│
                                              └──────────────────────────────┘
```

### Структура проекта

```text
metrics/
├── cmd/
│   ├── agent/              # точка входа агента
│   └── server/             # точка входа сервера
├── internal/
│   ├── agent/              # сбор метрик, worker pool, SendBatch
│   ├── compress/            # gzip-обёртки (Compress, Decompress)
│   ├── handler/             # HTTP-хендлеры и middleware
│   ├── model/               # структура Metrics (ID, Type, Value, Delta)
│   ├── persistence/         # файловое хранилище (atomic write)
│   ├── repository/
│   │   ├── memory/          # MemStorage с sync.Mutex
│   │   └── postgres/        # PsgsRepository на pgxpool
│   ├── retry/               # универсальный retry.Do с предикатом
│   └── sign/                # HMAC-SHA256 подпись (base64)
├── migrations/              # SQL-миграции + go:embed
└── .github/workflows/       # CI: metricstest, statictest
```

## 🚀 Быстрый старт

### 1. Клонирование

```bash
git clone https://github.com/BREJJNEVV/metrics.git
cd metrics
go mod download
```

### 2. Запуск сервера в памяти

```bash
go run ./cmd/server/main.go
```

### 3. Запуск агента

В отдельном терминале выполните:

```bash
go run ./cmd/agent/main.go
```

Агент начнёт собирать метрики и отправлять их на `http://localhost:8080`.

### 4. Проверка

```bash
curl http://localhost:8080/                  # HTML со списком метрик
curl http://localhost:8080/ping              # health check
curl http://localhost:8080/value/gauge/Alloc # значение конкретной метрики
```

## 🐘 Запуск с PostgreSQL

### 1. Создание пользователя и базы данных

Откройте `psql`:

```bash
psql -d postgres
```

Выполните SQL-команды:

```sql
CREATE USER metrics_app WITH PASSWORD 'AppPassword123';
CREATE DATABASE metrics OWNER metrics_app;
\q
```

### 2. Запуск сервера с DSN

```bash
go run ./cmd/server/main.go \
  -d="postgres://metrics_app:AppPassword123@localhost:5432/metrics?sslmode=disable"
```

Миграции применятся автоматически при старте.

### 3. Запуск агента

```bash
go run ./cmd/agent/main.go
```

## ⚙️ Конфигурация

Приоритет параметров: **переменная окружения → флаг → значение по умолчанию**.

### Сервер

| Флаг | Переменная окружения | По умолчанию | Описание |
|---|---|---|---|
| `-a` | `ADDRESS` | `localhost:8080` | Адрес эндпоинта |
| `-i` | `STORE_INTERVAL` | `300` | Интервал сохранения (`0` = синхронно) |
| `-f` | `FILE_STORAGE_PATH` | `""` | Путь к файлу хранилища |
| `-r` | `RESTORE` | `false` | Восстанавливать ли данные при старте |
| `-d` | `DATABASE_DSN` | `""` | DSN для PostgreSQL |
| `-k` | `KEY` | `""` | Ключ для HMAC-подписи |

### Агент

| Флаг | Переменная окружения | По умолчанию | Описание |
|---|---|---|---|
| `-a` | `ADDRESS` | `localhost:8080` | Адрес сервера |
| `-r` | `REPORT_INTERVAL` | `10` | Интервал отправки, секунд |
| `-p` | `POLL_INTERVAL` | `2` | Интервал опроса, секунд |
| `-k` | `KEY` | `""` | Ключ для HMAC-подписи |
| `-l` | `RATE_LIMIT` | `1` | Количество параллельных воркеров |

### Примеры

Сервер с PostgreSQL и ключом:

```bash
go run ./cmd/server/main.go \
  -d="postgres://..." \
  -k="secret"
```

Агент с интервалами 1 и 5 секунд и тремя воркерами:

```bash
go run ./cmd/agent/main.go -r=5 -p=1 -l=3 -k="secret"
```

Запуск через переменные окружения:

```bash
ADDRESS="localhost:9090" KEY="secret" go run ./cmd/server/main.go
```

## 🔌 API

### Приём метрики (текстовый формат)

```http
POST /update/<type>/<name>/<value>
Content-Type: text/plain
```

### Приём метрики (JSON)

```http
POST /update
Content-Type: application/json

{
  "id": "Alloc",
  "type": "gauge",
  "value": 123.45
}
```

### Батчевый приём метрик

```http
POST /updates
Content-Type: application/json
Content-Encoding: gzip
HashSHA256: <base64>

[
  {"id": "Alloc", "type": "gauge", "value": 123.45},
  {"id": "PollCount", "type": "counter", "delta": 10}
]
```

### Получение метрики

Текстовый формат:

```http
GET /value/<type>/<name>
```

Ответ: `text/plain`, например `123.45`.

JSON-формат:

```http
POST /value
Content-Type: application/json

{"id": "Alloc", "type": "gauge"}
```

Пример ответа:

```json
{
  "id": "Alloc",
  "type": "gauge",
  "value": 123.45
}
```

### Список метрик

```http
GET /
```

Ответ: `text/html` — страница со всеми метриками.

### Проверка состояния сервера

```http
GET /ping
```

Ответ: `200 OK` или `500 Internal Server Error`.

## 🔐 Подпись HMAC

Если задан параметр `-k` или переменная окружения `KEY`, тела запросов подписываются с помощью HMAC-SHA256:

```text
HashSHA256 = base64(hmac_sha256(key, body))
```

Агент добавляет заголовок `HashSHA256` к каждому запросу. Сервер проверяет подпись и возвращает `400 Bad Request`, если подпись отсутствует или неверна.

Сервер также подписывает ответы тем же ключом.

## 🛠️ Технологии

| Категория | Библиотека / технология |
|---|---|
| HTTP-роутер | `go-chi/chi` |
| PostgreSQL | `jackc/pgx/v5` |
| Миграции | `golang-migrate/migrate` |
| Коды ошибок PostgreSQL | `jackc/pgerrcode` |
| Системные метрики | `shirou/gopsutil` |
| Конфигурация | `caarlos0/env` |
| Логирование | `uber-go/zap` |
| Сжатие | Стандартный пакет `compress/gzip` |
| Криптография | Стандартные пакеты `crypto/hmac`, `crypto/sha256`, `encoding/base64` |
| Миграции (`embed`) | Стандартный пакет `embed` |

## 🧭 Ключевые технические решения

### 1. Инверсия зависимостей (DI)

Сервер работает с интерфейсом `Repository`, агент — с `MetricsReader` и `MetricsWriter`. Это позволяет подменять хранилища (память, файл, БД) и использовать моки в тестах.

### 2. Потокобезопасность

`MemStorage` защищён `sync.Mutex`. Методы `Gauges()` и `Counters()` возвращают копии `map`, чтобы внешний код не мог изменить внутреннее состояние хранилища.

### 3. Атомарная запись файла

Данные записываются во временный файл `metrics.json.tmp`, затем вызывается `os.Rename`. Это защищает от появления частично записанных данных при сбое.

### 4. Транзакции для батчей

`UpdateBatch` в PostgreSQL использует `Begin`, `Commit` и `Rollback`. Retry оборачивает всю транзакцию, а не отдельные вставки.

### 5. Универсальный retry

`retry.Do(ctx, isRetriable, fn)` — единая точка повторных попыток. Для агента предикат проверяет сетевые ошибки, для БД — SQLSTATE классов `08` и `40`.

### 6. Embed-миграции

SQL-файлы встраиваются в бинарник через `go:embed`. Деплой — один файл без внешних зависимостей для файлов миграций.

### 7. Порядок middleware

```text
WithLogging → GzipDecompress → GzipCompress → HashSign → HashVerify → handler
```

`HashSign` выполняется до `GzipCompress`, чтобы подписывать несжатое тело. `HashVerify` расположен после `GzipDecompress`, чтобы проверять распакованное тело.

### 8. Worker pool

`N` воркеров читают задачи из канала `jobs`. Параметр `-l` задаёт `N`. Задачи формируются в `main` через `CollectBatch` и отправляются в канал.

### 9. Graceful shutdown

Горутины слушают `ctx.Done()`. Главная функция ждёт `<-ctx.Done()` и завершает работу. При необходимости можно использовать `signal.NotifyContext` для обработки `SIGINT` и `SIGTERM`.

### 10. Изоляция middleware для подписи

`signResponseWriter` буферизует ответ в памяти, чтобы подписать его до сжатия. После вычисления хеша буфер отправляется в исходный `ResponseWriter`.
