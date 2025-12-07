# Metrics Collection Service

Сервис сбора и хранения метрик, состоящий из двух компонентов: **агента** (agent) для сбора метрик и **сервера** (server) для их хранения и обработки.

## Возможности

### Агент (Agent)

- **Сбор системных метрик**: CPU, память (Alloc, TotalAlloc, Sys и др.), статистика GC
- **Дополнительные метрики**: использование CPU и памяти через `gopsutil`
- **Отправка метрик**: поддержка HTTP и gRPC протоколов
- **Rate limiting**: ограничение параллельных запросов при отправке
- **Шифрование**: поддержка асимметричного шифрования данных (RSA)
- **Подпись данных**: HMAC SHA256 для проверки целостности
- **Конфигурация**: через флаги командной строки, переменные окружения или JSON файл

### Сервер (Server)

- **Приём метрик**: через HTTP REST API или gRPC
- **Хранение данных**:
  - In-memory хранилище
  - Файловое хранилище с настраиваемым интервалом синхронизации
  - PostgreSQL база данных с автоматическими миграциями
- **Безопасность**:
  - Проверка подписи данных (HMAC SHA256)
  - Асимметричное шифрование (RSA)
  - Фильтрация по IP подсети (trusted subnet)
- **Аудит**: логирование операций в файл или отправка на внешний сервер
- **Graceful shutdown**: корректное завершение работы по сигналам SIGINT, SIGTERM, SIGQUIT
- **Конфигурация**: через флаги командной строки, переменные окружения или JSON файл

## Установка

```bash
# Клонирование репозитория
git clone https://github.com/Pro100x3mal/go-musthave-metrics.git
cd go-musthave-metrics

# Установка зависимостей
go mod download

# Сборка
go build -o cmd/server/server ./cmd/server
go build -o cmd/agent/agent ./cmd/agent

# Или запуск через go run
go run ./cmd/server/main.go
go run ./cmd/agent/main.go
```

## Быстрый старт

### Запуск сервера (HTTP по умолчанию)

```bash
# Простой запуск с настройками по умолчанию
./cmd/server/server
# или через go run
go run ./cmd/server/main.go

# С указанием адреса и базы данных
./cmd/server/server -a localhost:8080 -d "postgres://user:pass@localhost/metrics?sslmode=disable"

# С файловым хранилищем
./cmd/server/server -a localhost:8080 -f /tmp/metrics.json -i 10 -r
```

### Запуск агента (HTTP по умолчанию)

```bash
# Простой запуск
./cmd/agent/agent -a localhost:8080
# или через go run
go run ./cmd/agent/main.go -a localhost:8080

# С настройкой интервалов
./cmd/agent/agent -a localhost:8080 -p 5 -r 15 -l 10
```

### Запуск с gRPC

```bash
# Сервер с gRPC (вместо HTTP)
./cmd/server/server -g localhost:3200

# Агент с gRPC (вместо HTTP)
./cmd/agent/agent -g localhost:3200
```

## Конфигурация

### Сервер

#### Параметры конфигурации

| Флаг | Переменная окружения | Описание | Значение по умолчанию |
|------|---------------------|----------|----------------------|
| `-a` | `ADDRESS` | Адрес HTTP сервера | `localhost:8080` |
| `-g` | `GRPC_ADDRESS` | Адрес gRPC сервера (если указан, HTTP не запускается) | - |
| `-l` | `LOG_LEVEL` | Уровень логирования (debug, info, warn, error) | `info` |
| `-i` | `STORE_INTERVAL` | Интервал сохранения метрик в файл (секунды) | `300` |
| `-f` | `FILE_STORAGE_PATH` | Путь к файлу для хранения метрик | - |
| `-r` | `RESTORE` | Восстановить метрики из файла при старте | `false` |
| `-d` | `DATABASE_DSN` | DSN для подключения к PostgreSQL | - |
| `-k` | `KEY` | Ключ для подписи данных (HMAC SHA256) | - |
| `-crypto-key` | `CRYPTO_KEY` | Путь к приватному ключу для расшифровки | - |
| `-t` | `TRUSTED_SUBNET` | Доверенная подсеть в CIDR формате | - |
| `-audit-file` | `AUDIT_FILE` | Путь к файлу аудита | `audit.json` |
| `-audit-url` | `AUDIT_URL` | URL для отправки событий аудита | - |
| `-c`, `-config` | `CONFIG` | Путь к JSON файлу конфигурации | - |

#### JSON конфигурация

```json
{
  "address": "localhost:8080",
  "grpc_address": "localhost:3200",
  "log_level": "info",
  "store_interval": "300s",
  "file_storage_path": "/tmp/metrics.json",
  "restore": true,
  "database_dsn": "postgres://user:pass@localhost/metrics",
  "signing_key": "secret",
  "crypto_key": "/path/to/private.key",
  "trusted_subnet": "192.168.1.0/24",
  "audit_file": "audit.json",
  "audit_url": "http://audit-server:8081/events"
}
```

### Агент

#### Параметры конфигурации

| Флаг | Переменная окружения | Описание | Значение по умолчанию |
|------|---------------------|----------|----------------------|
| `-a` | `ADDRESS` | Адрес HTTP сервера | `localhost:8080` |
| `-g` | `GRPC_ADDRESS` | Адрес gRPC сервера (если указан, HTTP не используется) | - |
| `-p` | `POLL_INTERVAL` | Интервал опроса метрик (секунды) | `2` |
| `-r` | `REPORT_INTERVAL` | Интервал отправки метрик (секунды) | `10` |
| `-l` | `RATE_LIMIT` | Лимит параллельных запросов при отправке | `5` |
| `-log-level` | `LOG_LEVEL` | Уровень логирования (debug, info, warn, error) | `info` |
| `-k` | `KEY` | Ключ для подписи данных (HMAC SHA256) | - |
| `-crypto-key` | `CRYPTO_KEY` | Путь к публичному ключу для шифрования | - |
| `-c`, `-config` | `CONFIG` | Путь к JSON файлу конфигурации | - |

#### JSON конфигурация

```json
{
  "address": "localhost:8080",
  "grpc_address": "localhost:3200",
  "poll_interval": "2s",
  "report_interval": "10s",
  "rate_limit": 5,
  "log_level": "info",
  "signing_key": "secret",
  "crypto_key": "/path/to/public.key"
}
```

## Приоритет настроек

Конфигурация применяется в следующем порядке (последующие перезаписывают предыдущие):

1. Значения по умолчанию
2. JSON файл конфигурации (`-c` / `-config` / `CONFIG`)
3. Флаги командной строки
4. Переменные окружения

## Логирование

Используется структурированное логирование с помощью `zap`:

- **debug**: детальная информация для отладки
- **info**: информационные сообщения (по умолчанию)
- **warn**: предупреждения
- **error**: ошибки

### Генерация protobuf

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/proto/metrics.proto
```

## Тестирование

```bash
# Запуск всех тестов
go test ./...

# С покрытием
go test -cover ./...

# Конкретный пакет
go test ./internal/server/handlers
```

## Примеры использования

### Пример 1: In-memory сервер с HTTP

```bash
# Сервер
./cmd/server/server -a localhost:8080

# Агент
./cmd/agent/agent -a localhost:8080 -p 2 -r 10
```

### Пример 2: Сервер с PostgreSQL и gRPC

```bash
# Сервер
export DATABASE_DSN="postgres://metrics:password@localhost:5432/metrics?sslmode=disable"
./cmd/server/server -g localhost:3200

# Агент
./cmd/agent/agent -g localhost:3200 -p 5 -r 15
```

### Пример 3: Файловое хранилище с шифрованием

```bash
# Генерация ключей
openssl genrsa -out private.key 4096
openssl rsa -in private.key -pubout -out public.key

# Сервер
./cmd/server/server -a localhost:8080 \
  -f /tmp/metrics.json \
  -i 60 \
  -r \
  -crypto-key private.key \
  -k "my-secret-key"

# Агент
./cmd/agent/agent -a localhost:8080 \
  -crypto-key public.key \
  -k "my-secret-key"
```

### Пример 4: С фильтрацией по подсети и аудитом

```bash
# Сервер
./cmd/server/server -a localhost:8080 \
  -t "192.168.1.0/24" \
  -audit-file /var/log/metrics-audit.json \
  -d "postgres://user:pass@localhost/metrics"

# Агент
./cmd/agent/agent -a localhost:8080
```

### Пример 5: Конфигурация через JSON файл

**server-config.json:**
```json
{
  "grpc_address": "localhost:3200",
  "log_level": "debug",
  "database_dsn": "postgres://metrics:password@localhost:5432/metrics",
  "trusted_subnet": "10.0.0.0/8",
  "audit_file": "audit.log"
}
```

**agent-config.json:**
```json
{
  "grpc_address": "localhost:3200",
  "poll_interval": "5s",
  "report_interval": "20s",
  "rate_limit": 10,
  "log_level": "debug"
}
```

```bash
# Запуск
./cmd/server/server -c server-config.json
./cmd/agent/agent -c agent-config.json
```
