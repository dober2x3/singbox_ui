# Встроенные API sing-box (Experimental)

> Официальная документация: https://sing-box.sagernet.org/configuration/experimental/

Sing-box предоставляет три встроенных API через секцию `experimental` в конфиге.
**Важно:** эти API собираются в бинарь sing-box только при включении соответствующих build-тегов
(кроме Clash API — он включён по умолчанию).

---

## Cache File

С версии 1.8.0. Хранит кеш на диске: fakeip, DNS-кеш, состояние выбранных нод, режим Clash.

### Структура

```json
{
  "cache_file": {
    "enabled": true,
    "path": "cache.db",
    "cache_id": "",
    "store_fakeip": false,
    "store_rdrc": false,
    "rdrc_timeout": "7d",
    "store_dns": false
  }
}
```

### Поля

| Поле | Тип | По умолчанию | Описание |
|------|-----|-------------|----------|
| `enabled` | bool | — | Включить кеш-файл |
| `path` | string | `"cache.db"` | Путь к файлу кеша |
| `cache_id` | string | `""` | Идентификатор в кеш-файле. Если не пусто — данные конфига хранятся в отдельном ключе |
| `store_fakeip` | bool | `false` | Сохранять fakeip в кеш-файл |
| `store_rdrc` | bool | `false` | ⚠️ Deprecated в 1.14.0. Кеш rejected DNS response |
| `rdrc_timeout` | duration | `"7d"` | Таймаут кеша rejected DNS response |
| `store_dns` | bool | `false` | 🆕 с 1.14.0. Хранить DNS-кеш в файле |

---

## Clash API

HTTP API, совместимый с Clash. Позволяет управлять прокси, правилами, смотреть логи и трафик
в реальном времени через REST + WebSocket.

### Структура

```json
{
  "clash_api": {
    "external_controller": "127.0.0.1:9090",
    "external_ui": "",
    "external_ui_download_url": "",
    "external_ui_download_detour": "",
    "secret": "",
    "default_mode": "Rule",
    "access_control_allow_origin": [],
    "access_control_allow_private_network": false
  }
}
```

### Поля

| Поле | Тип | По умолчанию | Описание |
|------|-----|-------------|----------|
| `external_controller` | string | — | **Адрес REST API.** Если пусто — Clash API отключён |
| `external_ui` | string | `""` | Путь к папке со статическим веб-интерфейсом (Yacd, Metacubexd). Отдаётся на `/ui` |
| `external_ui_download_url` | string | `"https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip"` | URL ZIP-архива с UI для авто-загрузки |
| `external_ui_download_detour` | string | `""` | Тег outbound'а для скачивания UI |
| `secret` | string | `""` | **Секретный ключ.** Передаётся в заголовке `Authorization: Bearer ${secret}`. **Обязателен**, если controller слушает на `0.0.0.0` |
| `default_mode` | string | `"Rule"` | Режим по умолчанию. Можно использовать в роутинге через правило `clash_mode` |
| `access_control_allow_origin` | string[] | `["*"]` | 🆕 с 1.10.0. CORS — разрешённые источники |
| `access_control_allow_private_network` | bool | `false` | 🆕 с 1.10.0. Разрешить доступ из частной сети |

### ⚠️ Deprecated поля (перенесены в cache_file)

| Поле | Замена |
|------|--------|
| `store_mode` | `cache_file.enabled` (включено по умолчанию) |
| `store_selected` | `cache_file.enabled` |
| `store_fakeip` | `cache_file.store_fakeip` |
| `cache_file` | `cache_file.enabled` + `cache_file.path` |
| `cache_id` | `cache_file.cache_id` |

### Эндпоинты Clash API

Ниже приведены основные эндпоинты Clash API.

#### Прокси

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/proxies` | Список всех прокси-групп + нод |
| `GET` | `/proxies/{name}` | Детальная информация о группе/ноде |
| `PUT` | `/proxies/{name}` | Переключить выбранную ноду в группе |
| `GET` | `/proxies/{name}/delay` | Задержка до ноды (TCP ping) |

#### Правила

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/rules` | Список правил маршрутизации |

#### Логи

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/logs` | Логи (WebSocket — реальное время) |
| `GET` | `/logs/level` | Получить текущий уровень логов |
| `PUT` | `/logs/level` | Установить уровень логов |

#### Трафик

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/traffic` | Трафик (WebSocket — реальное время) |
| `GET` | `/memory` | Потребление памяти (WebSocket) |

#### Общее

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/version` | Версия sing-box |
| `GET` | `/configs` | Текущий конфиг |
| `PUT` | `/configs` | Обновить конфиг |
| `GET` | `/connections` | Активные соединения |
| `DELETE` | `/connections` | Закрыть все соединения |
| `DELETE` | `/connections/{id}` | Закрыть конкретное соединение |
| `GET` | `/group` | Информация по всем группам |
| `GET` | `/group/{name}` | Информация по группе |
| `PUT` | `/group/{name}` | Переключить ноду в группе |

---

## V2Ray API

gRPC API для статистики трафика. **Не включён в сборку по умолчанию** — требуется
сборка с build-тегом `with_v2ray_api`.

### Структура

```json
{
  "v2ray_api": {
    "listen": "127.0.0.1:8080",
    "stats": {
      "enabled": true,
      "inbounds": ["socks-in"],
      "outbounds": ["proxy", "direct"],
      "users": ["sekai"]
    }
  }
}
```

### Поля

| Поле | Тип | Описание |
|------|-----|----------|
| `listen` | string | gRPC адрес. Если пусто — V2Ray API отключён |
| `stats.enabled` | bool | Включить сбор статистики |
| `stats.inbounds` | string[] | Список inbound'ов для учёта трафика |
| `stats.outbounds` | string[] | Список outbound'ов для учёта трафика |
| `stats.users` | string[] | Список пользователей для учёта трафика |

V2Ray API реализует gRPC-сервис `v2ray.core.app.stats.command.StatsService`:

```protobuf
service StatsService {
  rpc GetStats(GetStatsRequest) returns (GetStatsResponse);
  rpc QueryStats(QueryStatsRequest) returns (QueryStatsResponse);
  rpc GetSysStats(GetSysStatsRequest) returns (GetSysStatsResponse);
}
```

---

## Debug (pprof)

Встроенный дебаг-сервер на основе `net/http/pprof`.

### Структура

```json
{
  "debug": {
    "listen": "127.0.0.1:9999",
    "gc_percent": 100,
    "max_stack": 100,
    "max_threads": 10,
    "panic_on_fault": false,
    "trace_back": "single",
    "memory_limit": 0,
    "oom_killer": true
  }
}
```

### Поля

| Поле | Тип | По умолчанию | Описание |
|------|-----|-------------|----------|
| `listen` | string | — | Адрес pprof-сервера. Если пусто — Debug отключён |
| `gc_percent` | int | `100` | `GOGC` — порог сборки мусора в процентах |
| `max_stack` | int | `100` | Максимальный размер стека горутины |
| `max_threads` | int | `10` | Ограничение на количество потоков ОС |
| `panic_on_fault` | bool | `false` | Паниковать при fault'е памяти |
| `trace_back` | string | `"single"` | Режим трейса (`"single"`, `"all"`, `"system"`, `"crash"`) |
| `memory_limit` | int | `0` | Лимит памяти (байт, `0` = без лимита) |
| `oom_killer` | bool | `true` | Включить OOM Killer |

Доступные эндпоинты (стандартные pprof):

| Путь | Описание |
|------|----------|
| `GET /debug/pprof/` | Список профилей |
| `GET /debug/pprof/allocs` | Профиль аллокаций |
| `GET /debug/pprof/block` | Профиль блокировок |
| `GET /debug/pprof/goroutine` | Дамп горутин |
| `GET /debug/pprof/heap` | Профиль памяти (heap) |
| `GET /debug/pprof/mutex` | Профиль мьютексов |
| `GET /debug/pprof/profile` | CPU профиль (30s) |
| `GET /debug/pprof/threadcreate` | Профиль потоков |
| `GET /debug/pprof/trace` | Трассировка (1s) |

---

## Использование в проекте singbox-ui

В текущей реализации проекта **ни одно из этих API не используется**:

- Управление sing-box осуществляется через **Docker API** (запуск/остановка/логи)
- Конфиг пишется на диск и передаётся флагом `-c`
- Нет UI для настройки секции `experimental`
- `getFullConfig()` в store не включает `experimental` в финальный config.json

Типы `ClashAPIOptions`, `V2RayAPIOptions`, `CacheFileOptions`, `DebugOptions` определены
в `frontend/lib/store/singbox-config.ts` для полноты спецификации, но остаются
**неиспользованными** — потенциальная точка для будущего развития.

---

*Источник: https://sing-box.sagernet.org/configuration/experimental/*
