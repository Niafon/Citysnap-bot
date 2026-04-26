# CitySnap Bot

Telegram-бот для знакомств с механикой «фото дня» — загрузи фото, и его увидят все пользователи твоего города в течение 24 часов.

## Стек

- **Go 1.22+** — язык
- **PostgreSQL 16** — основная БД
- **Redis 7** — кэш, FSM, счётчики
- **pgx v5** — драйвер Postgres
- **go-telegram/bot** — Telegram Bot API
- **golang-migrate** — миграции БД
- **Docker** — контейнеризация

## Быстрый старт

```bash
# 1. Скопировать переменные окружения
cp .env.example .env
# Вписать TELEGRAM_TOKEN от @BotFather

# 2. Поднять инфраструктуру + применить миграции
make setup

# 3. Запустить бота
make run
```

## Команды

```bash
make run           # запуск
make test          # тесты с покрытием и race detector
make lint          # golangci-lint
make docker-up     # поднять Postgres + Redis
make docker-down   # остановить
make migrate-up    # применить миграции
make migrate-down  # откатить последнюю
```

## Архитектура

Трёхуровневая (Clean Architecture):

```
cmd/bot/main.go          → точка входа, DI
internal/handler/        → Telegram handlers, FSM
internal/service/        → бизнес-логика + интерфейсы
internal/repository/     → PostgreSQL (pgx), Redis
internal/model/          → доменные структуры
```

## Health Check

```bash
curl http://localhost:8080/readyz
# {"postgres":"ok","redis":"ok"}
```
