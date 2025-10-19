#!/bin/bash

# Скрипт для нагрузочного тестирования сервера метрик

SERVER_URL="http://localhost:8080"
METRICS_COUNT=1000
REQUESTS_COUNT=10000
CONCURRENT=50
PAYLOAD_FILE=./profiles/test_metrics_payload.json

echo "=== Нагрузочное тестирование сервера метрик ==="
echo "URL: $SERVER_URL"
echo "Количество метрик: $METRICS_COUNT"
echo "Количество запросов: $REQUESTS_COUNT"
echo "Конкурентность: $CONCURRENT"
echo ""

# Функция для генерации JSON payload с метриками
generate_batch_payload() {
    local count=$1
    echo -n '['
    for ((i=0; i<count; i++)); do
        if [ $i -gt 0 ]; then
            echo -n ','
        fi

        if [ $((i % 2)) -eq 0 ]; then
            echo -n "{\"id\":\"gauge_$i\",\"type\":\"gauge\",\"value\":$((i * 15 / 10))}"
        else
            echo -n "{\"id\":\"counter_$i\",\"type\":\"counter\",\"delta\":$i}"
        fi
    done
    echo -n ']'
}

# Создаем временный файл с данными
echo "Генерация тестовых данных..."
generate_batch_payload $METRICS_COUNT > "$PAYLOAD_FILE"
echo "✓ Данные сгенерированы: $(wc -c < "$PAYLOAD_FILE") байт"
echo ""

# Предварительная загрузка данных
echo "Предварительная загрузка данных в хранилище..."
curl -s -X POST "$SERVER_URL/updates/" \
    -H "Content-Type: application/json" \
    -d @"$PAYLOAD_FILE" > /dev/null
echo "✓ Данные загружены"
echo ""

# Проверка наличия утилиты hey
if ! command -v hey &> /dev/null; then
    echo "Ошибка: Утилита 'hey' не установлена"
    echo "Установите её командой: go install github.com/rakyll/hey@latest"
    rm -f "$PAYLOAD_FILE"
    exit 1
fi

# Запуск нагрузочных тестов
echo "=== Начало нагрузочного тестирования ==="
echo ""

# Проверка доступности сервера
echo "Проверка доступности сервера..."
if ! curl -s "$SERVER_URL/ping" > /dev/null 2>&1; then
    echo "Ошибка: Сервер недоступен по адресу $SERVER_URL"
    echo "Убедитесь, что сервер запущен"
    echo "Команда для запуска сервера:"
    echo 'DATABASE_DSN="postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable" go run ./cmd/server'
    exit 1
fi
echo "✓ Сервер доступен"
echo ""

# Тест 1: Batch update метрик
echo "Тест 1: Batch update метрик (POST /updates/)"
hey -n $REQUESTS_COUNT -c $CONCURRENT -m POST \
    -H "Content-Type: application/json" \
    -D "$PAYLOAD_FILE" \
    "$SERVER_URL/updates/"
echo ""

# Тест 2: Получение списка всех метрик
echo "Тест 2: Получение списка всех метрик (GET /)"
hey -n $REQUESTS_COUNT -c $CONCURRENT \
    "$SERVER_URL/"
echo ""

# Очистка
rm -f "$PAYLOAD_FILE"

echo "=== Нагрузочное тестирование завершено ==="
echo ""
echo "Снять профиль памяти:"
echo "curl http://localhost:8080/debug/pprof/heap -o profiles/base.pprof"
