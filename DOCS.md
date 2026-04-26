# Документация кода CitySnap Bot

Структурированное описание всех модулей и файлов проекта.  
Версия: MVP v0.1 · Go 1.22+ · март 2026

---

## Структура проекта

```
citysnap-bot/
├── cmd/bot/main.go                      Точка входа, DI, graceful shutdown
├── go.mod                               Модуль и зависимости
├── docker-compose.yml                   Postgres + Redis для разработки
├── Dockerfile                           Multi-stage build (~15MB образ)
├── Makefile                             Команды: run, test, migrate, setup
├── README.md                            Краткое описание для разработчика
├── .env.example                         Пример переменных окружения
├── .gitignore                           Игнорируемые файлы
│
├── internal/
│   ├── model/                           Доменные сущности
│   │   ├── user.go                      User, UserInterest + методы
│   │   ├── photo.go                     DailyPhoto, PhotoReaction
│   │   ├── swipe.go                     Swipe, Match
│   │   └── user_test.go                 Table-driven тесты (16 кейсов)
│   │
│   ├── service/                         Бизнес-логика + интерфейсы
│   │   ├── interfaces.go                7 контрактов для repository
│   │   ├── user.go                      UserService
│   │   ├── swipe.go                     SwipeService с мэтчингом
│   │   └── daily_photo.go               DailyPhotoService + cleanup worker
│   │
│   ├── repository/                      Реализация доступа к данным
│   │   ├── user_repo.go                 PostgreSQL via pgx
│   │   ├── swipe_repo.go                Свайпы + проверка мэтча
│   │   ├── match_repo.go                Создание и поиск мэтчей
│   │   ├── photo_repo.go                Daily photos + HideExpired
│   │   ├── interest_repo.go             Интересы через CopyFrom
│   │   └── cache/photo_cache.go         Redis cache для фида
│   │
│   ├── handler/                         Telegram Bot API адаптер
│   │   ├── bot.go                       Регистрация handlers, /start, /help
│   │   ├── onboarding.go                FSM-шаги регистрации
│   │   ├── swipe.go                     /search, /matches, callbacks
│   │   ├── daily_photo.go               /snap, /feed, /mysnap
│   │   └── fsm/
│   │       ├── state.go                 Константы состояний
│   │       └── storage.go               Redis storage с TTL
│   │
│   ├── apperror/errors.go               Sentinel ошибки
│   ├── config/config.go                 Загрузка конфигурации из env
│   └── server/health.go                 HTTP /healthz, /readyz
│
└── migrations/                          golang-migrate SQL файлы
    ├── 000001_create_users.up.sql
    ├── 000001_create_users.down.sql
    ├── 000002_create_interests.up.sql
    ├── 000002_create_interests.down.sql
    ├── 000003_create_daily_photos.up.sql
    ├── 000003_create_daily_photos.down.sql
    ├── 000004_create_swipes.up.sql
    ├── 000004_create_swipes.down.sql
    ├── 000005_create_matches.up.sql
    ├── 000005_create_matches.down.sql
    ├── 000006_create_reactions.up.sql
    └── 000006_create_reactions.down.sql
```

---

## 1. Доменные модели (`internal/model/`)

### 1.1. user.go — User и UserInterest

**Назначение:** определяет основные сущности пользователя и его интересов. Это POGO-структуры (Plain Old Go Objects) без зависимостей от инфраструктуры.

**Структура User:**

| Поле | Тип | Описание | json/db tags |
|------|-----|----------|--------------|
| ID | uuid.UUID | Первичный ключ (gen_random_uuid в Postgres) | id / id |
| TelegramID | int64 | ID пользователя в Telegram (UNIQUE) | telegram_id |
| Nickname | string | Отображаемое имя (3-20 символов) | nickname |
| Age | int | Возраст (18-100) | age |
| Description | string | Описание профиля (до 500 символов) | description |
| PhotoFileID | string | file_id фотографии в Telegram | photo_file_id |
| City | string | Город (lowercase, нормализованный) | city |
| IsActive | bool | Активность профиля | is_active |
| CreatedAt | time.Time | Время регистрации | created_at |
| UpdatedAt | time.Time | Последнее обновление | updated_at |

**Методы:**

- `NormalizeCity()` — pointer receiver, мутирует `City` в нижний регистр без пробелов. Используется в onboarding для нормализации ввода: "  МОСКВА  " → "москва".

- `IsComplete() bool` — value receiver, проверяет что все обязательные поля заполнены. Возвращает true только если nickname != "", age >= 18, city != "" и photo_file_id != "". Используется как guard перед командами /search, /snap, /feed.

**Структура UserInterest:** связь пользователя с категорией интересов (hobby, music, film, book, game). UNIQUE constraint в БД на (user_id, category, value) предотвращает дубли.

### 1.2. photo.go — DailyPhoto и PhotoReaction

**DailyPhoto** — фото дня пользователя.

| Поле | Назначение |
|------|------------|
| UserID | Автор фото |
| City | Город автора (для фильтрации фида) |
| PhotoFileID | file_id Telegram |
| Caption | Опциональная подпись (до 500 символов) |
| ViewCount | Счётчик просмотров (инкрементируется async) |
| CreatedAt | Время загрузки |
| ExpiresAt | now() + 24h при создании |
| IsVisible | false после expire (cleanup worker) |

**Методы:**

- `TimeLeft() time.Duration` — возвращает оставшееся время до expire. Если уже истекло — возвращает 0 (не отрицательное значение).
- `IsExpired() bool` — `time.Now().After(ExpiresAt)`.

**PhotoReaction** — реакции 🔥❤️👋 на фото. UNIQUE(photo_id, user_id) — одна реакция от одного пользователя.

### 1.3. swipe.go — Swipe и Match

**Swipe** — лайк/дизлайк/суперлайк одного пользователя в адрес другого. UNIQUE(swiper_id, swiped_id) на уровне БД защищает от дублей.

**Match** — взаимный лайк. Создаётся в той же транзакции что и второй лайк. Поле `is_active` позволяет в будущем разорвать мэтч (unmatch).

### 1.4. user_test.go — Table-driven тесты

16 тест-кейсов:
- `TestUser_IsComplete`: 8 кейсов (все поля, отсутствие каждого, граничные значения возраста)
- `TestUser_NormalizeCity`: 5 кейсов (uppercase, пробелы, mixed, уже нормальный, пустой)
- `TestDailyPhoto_IsExpired`: 3 кейса (expired, not expired, just expired)

Используется конвенция `t.Run(tt.name, func(t *testing.T))` для именованных subtests — каждый кейс отображается отдельно в выводе `go test -v`.

---

## 2. Сервисный слой (`internal/service/`)

### 2.1. interfaces.go — Контракты

Здесь определяются интерфейсы которые потребляет бизнес-логика. Реализуются они в пакете `repository`. Это инверсия зависимостей: service не знает про PostgreSQL, он работает с абстракцией.

**Семь интерфейсов:**

- **UserRepository** — Create, FindByTelegramID, FindByID, Update, FindCandidates
- **InterestRepository** — BatchCreate, FindByUserID, FindCommon (общие интересы двух юзеров)
- **SwipeRepository** — Create, HasSwiped, FindMatch (поиск встречного лайка)
- **MatchRepository** — Create, FindByUserID
- **DailyPhotoRepository** — Create, FindActiveByUser, FindActiveByCity, FindByIDs, HideExpired, IncrementViews
- **PhotoCacheStore** — GetCityFeed, SetCityFeed, DeleteCityFeed (Redis)

### 2.2. user.go — UserService

**Поля:** `repo UserRepository` (зависимость через интерфейс).

**Методы:**

`Register(ctx, telegramID, nickname)` — идемпотентная регистрация. Сначала проверяет наличие пользователя через FindByTelegramID. Если уже есть — возвращает существующего без создания. Иначе создаёт нового с IsActive=true. Логирует факт регистрации через slog.

`GetByTelegramID(ctx, tgID)` — обёртка над repo для удобства handler-слоя.

`GetByID(ctx, id)` — поиск по UUID. Используется при показе мэтчей (нужно получить второго юзера).

`Update(ctx, user)` — обновление профиля. Используется на каждом шаге онбординга.

### 2.3. swipe.go — SwipeService

**Зависимости:** SwipeRepository, MatchRepository, UserRepository.

**Метод Swipe** — главная бизнес-логика мэтчинга:

```
1. Проверка дубля: HasSwiped(swiper, target) → если true → ErrAlreadySwiped
2. Создание свайпа: swipes.Create()
3. Если type не like/superlike → return nil (мэтч невозможен)
4. Поиск встречного: FindMatch(target, swiper) → проверяет 
   есть ли свайп типа like/superlike от target к swiper
5. Если встречный есть:
   - matches.Create() с user1=swiper, user2=target
   - return созданный Match
6. Иначе return nil (свайп сохранён, мэтча нет)
```

**Важно:** в текущей реализации свайп и проверка мэтча — отдельные запросы (не одна транзакция). При высокой конкурентности это может создать гонку: два пользователя одновременно лайкают друг друга, оба получат "no match". Для production нужна транзакция SwipeRepository.SwipeWithMatch (помечено как TODO для middle-уровня).

`GetCandidates(ctx, userID, city, limit)` — получить limit анкет из города userID, исключая уже свайпнутых. ORDER BY RANDOM().

`GetMatches(ctx, userID)` — список активных мэтчей. Двусторонняя связь user1_id или user2_id = userID.

### 2.4. daily_photo.go — DailyPhotoService

**Зависимости:** DailyPhotoRepository, PhotoCacheStore.

**Метод Create:**

```
1. Проверка одного активного фото:
   FindActiveByUser → если есть → ErrSnapActive
2. Создание DailyPhoto с ExpiresAt = now() + 24h
3. repo.Create()
4. cache.DeleteCityFeed(city) — инвалидация кэша
5. Логирование
```

**Метод GetCityFeed (Cache-Aside Pattern):**

```
1. cache.GetCityFeed(city) → массив UUID
2. Если cache hit: repo.FindByIDs(ids) → отфильтровать excludeUID → return
3. Если cache miss: 
   - repo.FindActiveByCity(city)
   - cache.SetCityFeed(city, ids) — записать в Redis (TTL 5 мин)
   - отфильтровать excludeUID
   - return
```

**Метод StartCleanupWorker** — фоновая горутина для auto-expire:

```go
ticker := time.NewTicker(5 * time.Minute)
for {
    select {
    case <-ticker.C:
        expired, _ := repo.HideExpired(ctx)
        for _, p := range expired {
            cache.DeleteCityFeed(ctx, p.City)
        }
    case <-ctx.Done():
        return  // graceful shutdown
    }
}
```

`HideExpired` использует `UPDATE ... RETURNING` — возвращает список затронутых фото для инвалидации кэша их городов.

**filterOut, uniqueCities** — приватные хелперы. filterOut исключает фото указанного userID из фида (не показываем своё фото в /feed). uniqueCities собирает уникальные города из списка фото для batch-инвалидации.

---

## 3. Слой данных (`internal/repository/`)

### 3.1. user_repo.go — UserRepo

Реализует UserRepository через `pgxpool.Pool`. Все методы принимают context.Context первым аргументом.

**Create:** INSERT с RETURNING id, created_at, updated_at — заполняет поля User после вставки.

**FindByTelegramID:** SELECT с обработкой `pgx.ErrNoRows` через `errors.Is`. Возвращает `(nil, nil)` если не найден (без ошибки) — это идиома Go для опционального результата.

**FindByID:** аналогично FindByTelegramID, но по UUID.

**Update:** UPDATE с обновлением `updated_at = now()`. Возвращает только error — Update не нуждается в данных обратно.

**FindCandidates:** ключевой запрос для свайпов. Условия:
- city = $1 (тот же город)
- id != $2 (не себя)
- is_active = true
- id NOT IN (SELECT swiped_id FROM swipes WHERE swiper_id = $2) — не свайпнутые
- ORDER BY RANDOM() LIMIT $3 — случайные limit штук

Использует partial index `idx_users_city_active` для эффективности.

### 3.2. swipe_repo.go — SwipeRepo

**Create:** INSERT swipe с RETURNING id, created_at.

**HasSwiped:** `SELECT EXISTS(...)` — возвращает bool. Эффективнее чем COUNT(*).

**FindMatch:** проверяет встречный like/superlike:
```sql
SELECT EXISTS(
    SELECT 1 FROM swipes
    WHERE swiper_id = $1 AND swiped_id = $2
      AND type IN ('like', 'superlike')
)
```
Параметры: $1 = target (тот, кого только что свайпнули), $2 = swiper (текущий пользователь). Если функция возвращает true — есть взаимный лайк → создаём мэтч.

### 3.3. match_repo.go — MatchRepo

**Create:** INSERT в matches с RETURNING.

**FindByUserID:** SELECT WHERE user1_id = $1 OR user2_id = $1 AND is_active = true. Возвращает все мэтчи где пользователь участвует с любой стороны.

### 3.4. photo_repo.go — DailyPhotoRepo

**Create:** INSERT daily_photos с RETURNING id, created_at.

**FindActiveByUser:** одно активное фото пользователя — `is_visible = true AND expires_at > now()`. LIMIT 1.

**FindActiveByCity:** все активные фото города. ORDER BY created_at DESC — свежие первыми.

**FindByIDs:** SELECT WHERE id = ANY($1). Принимает массив UUID. Используется при cache hit — кэш хранит ID, нужно получить полные данные.

**HideExpired (важный метод!):** 
```sql
UPDATE daily_photos
SET is_visible = false
WHERE expires_at < now() AND is_visible = true
RETURNING id, user_id, city, ...
```
RETURNING позволяет одним запросом скрыть фото И получить список затронутых. Используется в cleanup worker.

**IncrementViews:** простой UPDATE с view_count = view_count + 1. Атомарная операция в Postgres.

### 3.5. interest_repo.go — InterestRepo

**BatchCreate** использует `pool.CopyFrom` — самый эффективный способ массовой вставки в Postgres (PostgreSQL COPY protocol). На 10x быстрее чем множественные INSERT'ы.

```go
rows := make([][]any, len(interests))
for i, it := range interests {
    rows[i] = []any{it.UserID, it.Category, it.Value}
}
_, err := pool.CopyFrom(ctx,
    pgx.Identifier{"user_interests"},
    []string{"user_id", "category", "value"},
    pgx.CopyFromRows(rows))
```

**FindCommon** — общие интересы двух пользователей через INNER JOIN:
```sql
SELECT a.category, a.value
FROM user_interests a
INNER JOIN user_interests b
  ON a.category = b.category AND a.value = b.value
WHERE a.user_id = $1 AND b.user_id = $2
```

### 3.6. cache/photo_cache.go — PhotoCache (Redis)

**GetCityFeed:** GET `feed:{city}` → массив UUID через JSON unmarshal. Возвращает `nil, nil` при cache miss (ошибка `redis.Nil`).

**SetCityFeed:** SET с TTL 5 минут. Сериализация ID списка в JSON.

**DeleteCityFeed:** DEL для инвалидации при создании или expire фото.

---

## 4. Handler-слой (`internal/handler/`)

### 4.1. bot.go — BotHandler

**Структура:**
```go
type BotHandler struct {
    users  *service.UserService
    swipes *service.SwipeService
    photos *service.DailyPhotoService
    fsm    *fsm.Storage
}
```

**Register(b)** — точка регистрации всех handlers. Подключает /start, /profile, /help плюс делегирует registerSwipeHandlers и registerPhotoHandlers.

**DefaultHandler** — fallback handler для текстовых сообщений не покрытых конкретными командами. Получает текущее FSM-состояние и направляет в нужный step handler:

```
state == await_nickname  → handleNickname
state == await_age       → handleAge
state == await_city      → handleCity
state == await_photo     → handlePhoto
state == await_description → handleDescription
state == await_snap_photo  → handleSnapPhoto
state == await_snap_caption → handleSnapCaption
default                  → "Используй /help"
```

**HandleStart:** проверяет существующего юзера. Если профиль полный (IsComplete) — приветствует. Иначе вызывает Register, переключает state в await_nickname и просит ник.

**HandleProfile:** показывает анкету текущего юзера через SendPhoto с caption в Markdown формате.

**HandleHelp:** список всех команд бота.

### 4.2. onboarding.go — FSM-шаги

Пять методов, по одному на каждый шаг:

**handleNickname:** валидация длины 3-20, сохранение, переход в await_age.

**handleAge:** парсинг strconv.Atoi, валидация 18-100, переход в await_city.

**handleCity:** валидация 2-100 символов, NormalizeCity (lowercase + trim), переход в await_photo.

**handlePhoto:** проверка `msg.Photo != nil && len > 0`, берём `Photo[len-1].FileID` (максимальное разрешение), переход в await_description.

**handleDescription:** валидация ≤500 символов, переход в StateReady, поздравление с завершением регистрации.

Каждый шаг при ошибке валидации НЕ переключает state — пользователь повторяет тот же шаг.

### 4.3. swipe.go — Свайпы и мэтчи

**registerSwipeHandlers:** регистрирует /search, /matches и `RegisterHandlerMatchFunc` для callback queries начинающихся с `swipe:`.

**HandleSearch:** проверяет IsComplete, вызывает showNextCandidate.

**HandleSwipeCallback:** парсит CallbackData формата `swipe:type:uuid` (например `swipe:like:abc-123`):

```
1. AnswerCallbackQuery (убирает "часики" в Telegram)
2. swiper := users.GetByTelegramID(cb.From.ID)
3. match := swipes.Swipe(swiper.ID, targetID, type)
4. Если ErrAlreadySwiped → toast "Уже оценивал"
5. Если match != nil → notifyMatch(swiper, target)
6. showNextCandidate — следующая анкета
```

**showNextCandidate:** GetCandidates(limit=1) → если пусто → "В городе пусто" → если есть → SendPhoto с caption и keyboard `👎❤️⭐`.

**notifyMatch:** SendPhoto обоим пользователям с фото второго и кнопкой "💬 Написать ник" с URL `tg://user?id=X` (открывает чат напрямую).

**HandleMatches:** список мэтчей в виде нумерованного текста с никами и городами.

### 4.4. daily_photo.go — Фото дня

**registerPhotoHandlers:** /snap, /feed, /mysnap + callbacks `feed:` и `snap:no_caption`.

**HandleSnap:** переключает в state await_snap_photo, просит фото.

**handleSnapPhoto** (FSM step): сохраняет file_id в FSM data через SetData, переключает в await_snap_caption, показывает кнопку "Без подписи".

**handleSnapCaption** (FSM step): берёт текст как подпись, обрезает до 500 символов, вызывает finalizeSnap.

**HandleSnapNoCaption:** callback "Без подписи" → finalizeSnap с пустой подписью.

**finalizeSnap:**
```
1. Получить fileID из FSM data
2. photo := photos.Create(userID, city, fileID, caption)
3. Если ErrSnapActive → "У тебя уже есть активное"
4. fsm.Clear (очистить состояние)
5. "Опубликовано на 24 часа"
```

**HandleFeed:** GetCityFeed → если пусто → "Будь первым" → showFeedPhoto(idx=0).

**HandleFeedCallback:** парсит `feed:next:idx` или `feed:prev:idx`, циклическая пагинация (последняя→первая).

**showFeedPhoto:** SendPhoto с caption `@ник | 📍 Город | ⏳ 6ч | 👁 42` и keyboard `◀ 3/12 ▶`. Асинхронно (через `go func`) инкрементирует view counter.

**HandleMySnap:** показывает активное фото пользователя со статистикой (просмотры, оставшееся время).

### 4.5. fsm/state.go — Константы состояний

```go
type State string

const (
    StateIdle             State = "idle"
    StateAwaitNickname    State = "await_nickname"
    StateAwaitAge         State = "await_age"
    StateAwaitCity        State = "await_city"
    StateAwaitPhoto       State = "await_photo"
    StateAwaitDescription State = "await_description"
    StateReady            State = "ready"
    StateAwaitSnapPhoto   State = "await_snap_photo"
    StateAwaitSnapCaption State = "await_snap_caption"
)
```

### 4.6. fsm/storage.go — Redis FSM Storage

**Get/Set/Clear:** SET/GET/DEL ключа `fsm:{telegram_id}` с TTL 30 минут.

**SetData/GetData:** для промежуточных данных — `fsm:data:{tg_id}:{key}`. Используется для хранения file_id во время загрузки фото между шагами FSM.

При cache miss возвращает StateIdle (а не ошибку) — это значит "пользователь не в FSM".

---

## 5. Инфраструктура

### 5.1. cmd/bot/main.go — Точка входа

Полная DI-цепочка:

```
1. config.MustLoad() — загрузка env
2. setupLogger(env) — slog JSON в prod, text в dev
3. signal.NotifyContext(SIGTERM, SIGINT) — корневой ctx
4. pgxpool.New(DatabaseURL) + pool.Ping — Postgres
5. redis.NewClient + rdb.Ping — Redis
6. Создание repo → cache → service → fsm → handler
7. go photoSvc.StartCleanupWorker(ctx) — фоновый worker
8. http.Server :8080 для health checks
9. bot.New(token) + handler.Register(b)
10. b.Start(ctx) — блокирует до отмены
11. Graceful shutdown: healthSrv.Shutdown с таймаутом 10с,
    pool.Close, rdb.Close
```

### 5.2. config/config.go

Простая загрузка env-переменных через `os.Getenv`. Все переменные имеют дефолты для удобства dev. Production должен переопределить через `.env` или k8s secrets.

### 5.3. apperror/errors.go

Sentinel ошибки как переменные пакета:

```go
var (
    ErrUserNotFound  = errors.New("user not found")
    ErrAlreadySwiped = errors.New("already swiped this user")
    ErrSnapActive    = errors.New("active snap already exists")
    ErrPhotoExpired  = errors.New("photo has expired")
    ErrRateLimit     = errors.New("daily swipe limit exceeded")
    ErrInvalidInput  = errors.New("invalid input")
)
```

Проверка через `errors.Is(err, apperror.ErrAlreadySwiped)` работает по всей цепочке оборачиваний.

### 5.4. server/health.go

HTTP handler для `/readyz`. Пингует Postgres и Redis с таймаутом 2 секунды. Возвращает JSON `{"postgres":"ok","redis":"ok"}` со статусом 200, либо 503 если хоть один из них недоступен. `/healthz` всегда возвращает 200 (liveness — приложение запущено).

---

## 6. Миграции БД

Шесть миграций, по парам up/down:

**000001 users** — основная таблица пользователей с partial index `idx_users_city_active` для эффективного поиска кандидатов в свайпах.

**000002 user_interests** — связь пользователя с интересами. UNIQUE(user_id, category, value).

**000003 daily_photos** — фото дня. Composite index `idx_photos_feed (city, is_visible, created_at DESC) WHERE is_visible = true` для быстрого фида города.

**000004 swipes** — UNIQUE(swiper_id, swiped_id) защищает от дублей на уровне БД.

**000005 matches** — двусторонняя связь user1_id, user2_id.

**000006 photo_reactions** — UNIQUE(photo_id, user_id) — одна реакция от одного юзера.

Все таблицы используют `gen_random_uuid()` для PK и `TIMESTAMPTZ DEFAULT now()` для временных меток. ON DELETE CASCADE на foreign keys обеспечивает целостность при удалении пользователей.

---

## 7. Запуск и проверка

```bash
# Установка
unzip citysnap-bot-mvp.zip && cd citysnap-bot
cp .env.example .env  # вписать TELEGRAM_TOKEN
go mod tidy

# Поднять Postgres + Redis + применить миграции
make setup

# Запустить бота
make run

# Проверить health
curl http://localhost:8080/readyz
# → {"postgres":"ok","redis":"ok"}

# Тесты
make test
```

В Telegram открыть бота → `/start` → пройти онбординг → `/profile` (показать анкету) → `/snap` (загрузить фото) → `/feed` (посмотреть город) → `/search` (свайпы).

---

## 8. Метрики проекта

- Файлов кода: 22 .go + 12 .sql + 6 инфраструктурных = 40
- Строк кода: ~1500 (без тестов и комментариев)
- Тестов: 16 кейсов в model layer (table-driven)
- Зависимости: 4 прямые (pgx, go-redis, uuid, go-telegram/bot)
- Размер Docker-образа: ~15MB (multi-stage Alpine)
