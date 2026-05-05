# Real Time Chat (Go + WebSocket + Redis)

Минимальный WebSocket-чат на Go с JWT-аутентификацией и Redis Pub/Sub.

## Что умеет проект

- WebSocket endpoint: `GET /global`
- JWT-проверка перед апгрейдом до WebSocket
- Комнаты (`join` / `leave`)
- Рассылка `message` всем участникам комнаты
- Хранение истории сообщений в Redis
- Отправка последних сообщений при входе в комнату (`type: "history"`)

## Технологии

- Go
- [chi](https://github.com/go-chi/chi)
- [gorilla/websocket](https://github.com/gorilla/websocket)
- [go-redis/v9](https://github.com/redis/go-redis)
- Redis 7 (для Pub/Sub и истории)

## Быстрый старт (Docker)

```bash
docker compose up --build
```

После запуска:
- приложение: `localhost:8080`
- Redis: `localhost:6379`

Переменная окружения приложения:
- `REDIS_ADDR` (по умолчанию `localhost:6379`)

## Запуск локально (без Docker)

1) Подними Redis локально на `localhost:6379`  
2) Запусти приложение:

```bash
go run ./cmd
```

Сервер слушает `:8080`.

## Аутентификация (JWT)

Маршрут `/global` защищен middleware.

Ожидается заголовок:
```text
Authorization: Bearer <token>
```

Требования к payload токена:
- должен быть `user_id` (число)

В текущей версии подпись токена проверяется ключом `"secret"` (захардкожено в коде).

Пример payload:

```json
{
  "user_id": 1
}
```

## Подключение к WebSocket

URL:

```text
ws://localhost:8080/global?username=ars
```

Если `username` не передан, будет использовано `anonym`.

## Формат событий

Базовая структура:

```json
{
  "type": "join | leave | message | history",
  "room": "global",
  "text": "hello",
  "user_id": 1,
  "username": "ars",
  "history": []
}
```

### События от клиента

#### join
```json
{
  "type": "join",
  "room": "global"
}
```

#### leave
```json
{
  "type": "leave",
  "room": "global"
}
```

#### message
```json
{
  "type": "message",
  "room": "global",
  "text": "Привет!"
}
```

### События от сервера

#### message
Обычное сообщение комнаты (с заполненными `user_id` и `username`).

#### history
При входе в комнату сервер отправляет:

```json
{
  "type": "history",
  "room": "global",
  "history": [
    {
      "type": "message",
      "room": "global",
      "text": "старое сообщение",
      "user_id": 2,
      "username": "bob"
    }
  ]
}
```

`history` идёт в порядке от старых к новым.

## Как работает история в Redis

- Ключ: `chat:room:{room}:messages`
- При `message`: `LPUSH` + `LTRIM` (ограничение размера)
- Хранится максимум `200` сообщений (`historyListMax`)
- На вход в комнату отдается до `50` последних (`historyFetchSize`)

## Структура проекта

```text
cmd/main.go                    # entrypoint и роутинг
internal/middleware/           # JWT middleware
internal/ws/                   # websocket логика (hub, client, history, events)
Dockerfile
docker-compose.yml
```

## Полезные команды

```bash
go build ./...
docker compose up --build
docker compose down
```

## Ограничения текущей версии

- JWT secret захардкожен (`"secret"`)
- Нет отдельного health endpoint у приложения
- Нет персистентного хранилища пользователей/комнат (только runtime + Redis для сообщений)
