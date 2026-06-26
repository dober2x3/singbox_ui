# Clash API Integration Design

**Date:** 2026-06-11
**Status:** Draft
**Branch:** feat/clash-api-integration

## 1. Goal

Добавить в Go backend singbox_ui поддержку Clash API sing-box для runtime-операций:
переключение прокси, мониторинг трафика, просмотр соединений, логов, правил и смена
режима. Docker удаляется и заменяется заглушкой. Prober и Speedtest не изменяются.

## 2. Архитектура

Вертикальные срезы (Vertical Slices): каждая фича — сквозной слой от HTTP-роута до
Clash API запущенного sing-box процесса.

```
Frontend (Next.js) ─── не входит в это ТЗ, только бэкенд

Backend (Go/Gin):
  6 новых роутов + Config Editor (существующий) + Prober/Speedtest (существующий)
    │
    ├── Port Manager (shared)
    ├── Clash HTTP Client (shared)
    ├── WS/SSE Proxy (shared)
    │
    ▼
sing-box процессы (N штук, native, без Docker)
  default  :9090
  office   :9091
  home     :9092
  ...
```

### 2.1. Docker — заглушка

| Файл | Действие |
|------|----------|
| `server/internal/singbox/runtime_docker.go` | Заменить на заглушку, возвращающую `ErrNotAvailable` |
| `server/internal/pkg/tunnelrunner/docker.go` | Удалить или закомментировать |
| `server/internal/singbox/runtime_native.go` | Оставить — единственная реализация Runtime |
| `server/internal/pkg/tunnelrunner/native.go` | Оставить — единственная реализация Runner |

### 2.2. NativeRuntime

Уже существует. Проверить:
- Остановка процесса через SIGTERM + timeout + SIGKILL
- Очистка PID-файлов при остановке
- Поддержка multi-instance по имени (PID-файлы по имени)

### 2.3. Prober и Speedtest

Без изменений. Prober делает TCP `net.Dial` к нодам независимо от sing-box.
Speedtest использует `tunnelrunner/native.go` (временные sing-box процессы).

## 3. Shared Infrastructure

### 3.1. Port Manager

**Пакет:** `server/internal/clashapi/portmanager.go`

```go
type PortManager struct {
    basePort int
    assigned map[string]int
    mu       sync.Mutex
}

func NewPortManager(basePort int) *PortManager
func (pm *PortManager) Assign(instanceName string) int       // 9090+N
func (pm *PortManager) Release(instanceName string)           // освободить порт
func (pm *PortManager) Get(instanceName string) (int, bool)   // получить назначенный порт
func (pm *PortManager) List() map[string]int                  // все назначения
func (pm *PortManager) Save(path string) error                // сохранить в JSON
func (pm *PortManager) Load(path string) error                // загрузить из JSON
```

- basePort = 9090
- Первый инстанс (default) → 9090, второй → 9091, и т.д.
- При удалении инстанса порт освобождается
- PortManager инициализируется при старте backend и живёт в синглтоне
- **Персистентность:** маппинг имя → порт сохраняется в JSON-файл
  (`{datadir}/clash_ports.json`). При перезапуске backend'а порты
  восстанавливаются, чтобы уже запущенные sing-box процессы не потеряли
  связь с назначенным портом Clash API.

### 3.2. Clash HTTP Client

**Пакет:** `server/internal/clashapi/client.go`

```go
type Client struct {
    baseURL string    // http://127.0.0.1:{port}
    secret  string    // Authorization: Bearer {secret}
}
```

Методы, сгруппированные по вертикалям:

**Proxies:**
```go
func (c *Client) GetProxies() (*ProxiesResponse, error)
func (c *Client) GetProxy(name string) (*ProxyGroup, error)
func (c *Client) SwitchProxy(groupName, proxyName string) error
func (c *Client) GetProxyDelay(name, url string, timeout int) (*DelayResponse, error)
```

**Traffic:**
```go
func (c *Client) StreamTraffic(ctx context.Context) (<-chan TrafficData, error)   // WS
func (c *Client) StreamMemory(ctx context.Context) (<-chan MemoryData, error)     // WS
```

**Connections:**
```go
func (c *Client) GetConnections() (*ConnectionsResponse, error)
func (c *Client) CloseAllConnections() error
func (c *Client) CloseConnection(id string) error
```

**Logs:**
```go
func (c *Client) StreamLogs(ctx context.Context, level string) (<-chan LogEntry, error)  // WS
func (c *Client) GetLogLevel() (string, error)
func (c *Client) SetLogLevel(level string) error
```

**Rules:**
```go
func (c *Client) GetRules() (*RulesResponse, error)
```

**Config/Mode:**
```go
func (c *Client) GetConfigs() (*ConfigResponse, error)
func (c *Client) PatchConfigs(partial map[string]interface{}) error
```

### 3.3. WS/SSE Proxy

**Пакет:** `server/internal/clashapi/stream.go`

Проксирует WebSocket-соединения от Clash API как Server-Sent Events для фронтенда.

```go
// StreamHandler создаёт Gin handler, который:
// 1. Открывает WS к Clash API (ws://127.0.0.1:{port}/{endpoint})
// 2. Читает сообщения из WS
// 3. Пишет их в gin.ResponseWriter как SSE (text/event-stream)
func (c *Client) StreamHandler(clashPort int, wsEndpoint string) gin.HandlerFunc
```

SSE формат:
```
data: {"up":1234,"down":5678}\n\n
```

Нужен для:
- `StreamTraffic` → SSE `/api/clash/instances/:name/traffic`
- `StreamMemory` → SSE `/api/clash/instances/:name/memory`
- `StreamLogs` → SSE `/api/clash/instances/:name/logs`

## 4. Backend Routes (6 вертикалей)

### 4.1. Proxies

Новый файл: `server/internal/clashapi/routes_proxies.go`

```go
func RegisterProxiesRoutes(rg *gin.RouterGroup, pm *PortManager)

// GET /api/clash/instances/:name/proxies
// PUT /api/clash/instances/:name/proxies/:group
// GET /api/clash/instances/:name/proxies/:group
// GET /api/clash/instances/:name/proxies/:group/delay?url=...&timeout=...
```

Каждый handler:
1. Извлекает `:name` из URL
2. Получает порт из PortManager
3. Создаёт Clash Client
4. Выполняет соответствующий метод
5. Возвращает результат

### 4.2. Traffic

Новый файл: `server/internal/clashapi/routes_traffic.go`

```go
func RegisterTrafficRoutes(rg *gin.RouterGroup, pm *PortManager)

// GET /api/clash/instances/:name/traffic   → SSE
// GET /api/clash/instances/:name/memory     → SSE
```

### 4.3. Connections

Новый файл: `server/internal/clashapi/routes_connections.go`

```go
func RegisterConnectionsRoutes(rg *gin.RouterGroup, pm *PortManager)

// GET    /api/clash/instances/:name/connections
// DELETE /api/clash/instances/:name/connections
// DELETE /api/clash/instances/:name/connections/:id
```

### 4.4. Logs

Новый файл: `server/internal/clashapi/routes_logs.go`

```go
func RegisterLogsRoutes(rg *gin.RouterGroup, pm *PortManager)

// GET /api/clash/instances/:name/logs           → SSE
// GET /api/clash/instances/:name/logs/level
// PUT /api/clash/instances/:name/logs/level
```

### 4.5. Rules

Новый файл: `server/internal/clashapi/routes_rules.go`

```go
func RegisterRulesRoutes(rg *gin.RouterGroup, pm *PortManager)

// GET /api/clash/instances/:name/rules
```

### 4.6. Mode

Новый файл: `server/internal/clashapi/routes_mode.go`

```go
func RegisterModeRoutes(rg *gin.RouterGroup, pm *PortManager)

// GET /api/clash/instances/:name/mode
// PUT /api/clash/instances/:name/mode  body: {"mode": "rule"}
```

### 4.7. Регистрация всех роутов

В `main.go` или в `internal/clashapi/register.go`:

```go
func RegisterAllRoutes(rg *gin.RouterGroup, pm *PortManager) {
    RegisterProxiesRoutes(rg, pm)
    RegisterTrafficRoutes(rg, pm)
    RegisterConnectionsRoutes(rg, pm)
    RegisterLogsRoutes(rg, pm)
    RegisterRulesRoutes(rg, pm)
    RegisterModeRoutes(rg, pm)
}
```

API группа: `/api/clash`

```go
api := r.Group("/api")
{
    // Существующие роуты
    singboxGroup := api.Group("/singbox")
    // ... (без изменений)

    // Новые Clash API роуты
    clashGroup := api.Group("/clash")
    clashapi.RegisterAllRoutes(clashGroup, portManager)
}
```

## 5. Config Editor — минимальные изменения

### 5.1. Добавить `experimental.clash_api` в `getFullConfig()`

В `frontend/lib/store/singbox-config.ts` (файл фронтенда, но изменение тривиальное):

```typescript
fullConfig.experimental = {
    clash_api: {
        external_controller: `127.0.0.1:${clashPort}`,
        secret: "",
        default_mode: "Rule",
    }
}
```

Бэкенд передаёт порт Clash API для каждого инстанса через
существующий эндпоинт загрузки конфига. В `GET /api/singbox/instances/:name/config`
добавить поле `clash_port` в ответ. Фронтенд использует его при генерации
config.json.

Это изменение нужно, чтобы sing-box при запуске включал Clash API.

## 6. Изменения в main.go

```go
func main() {
    // ... существующий код ...

    // Инициализация Port Manager
    portManager := clashapi.NewPortManager(9090)

    // Назначение портов для существующих инстансов
    for _, inst := range instances {
        portManager.Assign(inst.Name)
    }

    // Регистрация Clash API роутов
    api := r.Group("/api")
    clashGroup := api.Group("/clash")
    clashapi.RegisterAllRoutes(clashGroup, portManager)
    
    // Очистка портов при удалении инстанса
    // (в существующем handler удаления добавить portManager.Release(name))
}
```

## 7. Не входит в это ТЗ

- Frontend-компоненты (страницы, графики, таблицы — будут отдельно)
- Аутентификация/авторизация
- `PUT /configs` частичное обновление конфига
- Синхронизация состояний между редактором и Clash API
- Замена Prober на Clash API delay

## 8. Тестирование

- Port Manager: unit-тесты (назначение, освобождение, повторное назначение)
- Clash Client: интеграционные тесты с заглушкой HTTP-сервера, имитирующей Clash API
- Routes: тесты Gin handler (через `httptest.NewRecorder`)
- Smoke test: запустить sing-box с тестовым конфигом, проверить что Clash API отвечает
- Port Manager: тест персистентности (Save/Load, восстановление после перезапуска)

## 9. Структура пакета

```
server/internal/clashapi/
    portmanager.go         // Port Manager
    client.go              // Clash HTTP Client
    client_proxies.go      // Proxies методы клиента
    client_traffic.go      // Traffic методы клиента
    client_connections.go  // Connections методы клиента
    client_logs.go         // Logs методы клиента
    client_rules.go        // Rules методы клиента
    client_config.go       // Config/Mode методы клиента
    stream.go              // WS/SSE прокси
    routes_proxies.go      // Gin роуты Proxies
    routes_traffic.go      // Gin роуты Traffic
    routes_connections.go  // Gin роуты Connections
    routes_logs.go         // Gin роуты Logs
    routes_rules.go        // Gin роуты Rules
    routes_mode.go         // Gin роуты Mode
    register.go            // RegisterAllRoutes
    models.go              // Типы ответов Clash API
```

## 10. План реализации (оценка)

| Этап | Что делаем | Время |
|------|-----------|-------|
| 1 | Port Manager + Clash Client базовый + заглушка Docker | 2 дня |
| 2 | Вертикаль Proxies (прокси-роуты, get/switch/delay) | 2 дня |
| 3 | Вертикаль Connections (get/delete) | 1 день |
| 4 | Вертикаль Rules + Mode | 1 день |
| 5 | WS/SSE прокси + Вертикали Traffic + Logs | 2 дня |
| 6 | Регистрация роутов, правка main.go, интеграция | 1 день |
| 7 | Тесты, отладка | 2 дня |
| | **Итого** | **~11 дней** |
