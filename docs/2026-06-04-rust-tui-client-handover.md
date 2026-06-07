# Rust TUI Client — Handover Instructions

**Date:** 2026-06-04
**Author:** AI Agent
**Status:** Draft

## 1. Executive Summary

Создать TUI-клиент на Rust для управления sing-box через REST API. Клиент должен быть
полнофункциональной альтернативой веб-интерфейсу — управление подписками, экземплярами
sing-box, WARP, WireGuard, probing'ом, speedtest'ами и сертификатами.

**Подход:** генерация Rust-клиента из OpenAPI (Swagger 2.0), затем построение интерактивного
TUI на `ratatui` поверх готового клиента.

---

## 2. Архитектура

```
┌──────────────────────┐
│     Rust TUI App     │  ← ratatui + crossterm
│  (main.rs + views/)  │
├──────────────────────┤
│   API Client Layer   │  ← сгенерирован из swagger.json
│  (reqwest + serde)   │
├──────────────────────┤
│   REST API (Go)      │  ← singbox_ui backend, :8080/api
│  54 endpoints        │
└──────────────────────┘
```

### 2.1 Компоненты (все на Rust)

| Компонент | Технология | Назначение |
|-----------|-----------|------------|
| **TUI Framework** | `ratatui` (+ `crossterm`) | Рендер терминала, обработка клавиш |
| **HTTP Client** | `reqwest` | REST-запросы к бэкенду |
| **API Types** | `serde` + `serde_json` | Сериализация/десериализация всех моделей |
| **Task Runner** | `tokio` | Асинхронные запросы + polling |
| **State Management** | `tokio::sync::watch` + `mpsc` | Реактивное обновление UI |

---

## 3. Шаг 1: Генерация Rust API Client из OpenAPI

Исходный файл: `server/internal/docs/swagger.json` (Swagger 2.0, 54 endpoints, 38 type definitions).

### 3.1 Вариант A: openapi-generator (рекомендуется)

Самый надёжный способ для Swagger 2.0. Использует официальный [OpenAPI Generator](https://openapi-generator.tech/).

```bash
# Установка (через npm — не требует Java)
npm install @openapitools/openapi-generator-cli -g

# Или через Docker (если npm не подходит):
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/swagger.json \
  -g rust \
  -o /local/singbox-api-client
```

**Параметры генерации:**

| Параметр | Значение | Пояснение |
|----------|----------|-----------|
| `-g rust` | — | Генератор Rust |
| `--library reqwest` | reqwest | Использовать `reqwest` вместо `hyper` |
| `--additional-properties=packageName=singbox_api` | — | Имя crate'а |
| `--additional-properties=preferUnsignedInt=true` | — | unsigned vs i32 |
| `--skip-validate-spec` | — | Swagger 2.0 может не пройти валидацию OpenAPI 3 |

**Что сгенерируется:**

```
singbox-api-client/
├── Cargo.toml
├── src/
│   ├── lib.rs              # re-exports
│   ├── apis/
│   │   ├── mod.rs
│   │   ├── certificate_api.rs
│   │   ├── prober_api.rs
│   │   ├── singbox_api.rs
│   │   ├── speedtest_api.rs
│   │   ├── subscription_api.rs
│   │   ├── system_api.rs
│   │   ├── warp_api.rs
│   │   └── wireguard_api.rs
│   └── models/
│       ├── mod.rs
│       └── *.rs            # 38 структур
```

### 3.2 Вариант B: progenitor (более идиоматичный Rust)

[`progenitor`](https://github.com/oxidecomputer/progenitor) от Oxide Computer.
Генерирует эргономичный клиент с builder-паттерном.

```bash
cargo install progenitor
```

Использование в `build.rs`:

```rust
// build.rs
fn main() {
    println!("cargo:rerun-if-changed=swagger.json");
    let spec = include_str!("swagger.json");
    let mut gen = progenitor::Generator::default();
    let content = gen.generate_text(&serde_json::from_str(spec).unwrap()).unwrap();
    std::fs::write("src/api.rs", content).unwrap();
}
```

**Важно:** `progenitor` лучше всего работает с OpenAPI 3.x. Для Swagger 2.0 может
потребоваться конвертация через `swagger2openapi`:

```bash
npm install -g swagger2openapi
swagger2openapi swagger.json -o openapi.json
```

### 3.3 Вариант C: ручная имплементация (fallback)

Если генераторы не справляются — написать клиент вручную. Это ~600 строк типов
и ~400 строк HTTP-методов. Плюс: полный контроль, минус: трудозатраты.

**Структура:**

```
src/
├── api.rs               # HTTP-клиент (reqwest Client wrapper)
├── types.rs             # Все 38 структур + enum'ы
├── certificate.rs       # certificate endpoints
├── prober.rs            # prober endpoints
├── singbox.rs           # singbox endpoints
├── speedtest.rs         # speedtest endpoints
├── subscription.rs      # subscription endpoints
├── system.rs            # system endpoints
├── warp.rs              # warp endpoints
└── wireguard.rs         # wireguard endpoints
```

Пример одного модуля:

```rust
// types.rs
#[derive(Debug, Deserialize)]
pub struct StatusResponse {
    pub running: Option<bool>,
    pub config_ok: Option<bool>,
}

// api.rs
#[derive(Clone)]
pub struct ApiClient {
    client: reqwest::Client,
    base_url: String,
}

impl ApiClient {
    pub fn new(base_url: &str) -> Self { ... }
}

// singbox.rs
impl ApiClient {
    pub async fn get_status(&self) -> Result<StatusResponse, Error> {
        self.client.get(format!("{}/singbox/status", self.base_url))
            .send().await?
            .json().await
            .map_err(Into::into)
    }
}
```

---

## 4. Шаг 2: TUI Design Specification

### 4.1 Стек зависимостей (Cargo.toml)

```toml
[package]
name = "singbox-tui"
version = "0.1.0"
edition = "2024"

[dependencies]
# TUI
ratatui = "0.29"
crossterm = "0.28"

# HTTP + JSON
reqwest = { version = "0.12", features = ["json"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tokio = { version = "1", features = ["full"] }

# CLI
clap = { version = "4", features = ["derive"] }
tracing = "0.1"
tracing-subscriber = "0.3"
anyhow = "1"
thiserror = "2"

# Optional: generated client crate
singbox_api = { path = "../singbox-api-client" }
```

### 4.2 Структура проекта

```
src/
├── main.rs              # Entry point, CLI args, app loop
├── app.rs               # App state, event loop, dispatch
├── api.rs               # re-export клиента (generated или manual)
├── types.rs             # re-export типов (generated или manual)
├── event.rs             # Event loop (keyboard, polling, tick)
├── ui/
│   ├── mod.rs           # Layout root
│   ├── dashboard.rs     # Главная панель — статус, health
│   ├── instances.rs     # Экземпляры sing-box (список + управление)
│   ├── instance_detail.rs # Логи, статус, конфиг конкретного экземпляра
│   ├── subscriptions.rs # Список подписок
│   ├── subscription_detail.rs # Ноды подписки
│   ├── prober.rs        # Prober status + results
│   ├── speedtest.rs     # Speed test
│   ├── warp.rs          # WARP account + registration
│   ├── wireguard.rs     # WireGuard keys + configs
│   ├── certificate.rs   # Certificate management
│   └── common.rs        # Shared widgets (spinner, table, form)
```

### 4.3 Навигация и Layout

```
┌────────────────────────────────────────────┐
│  SingBox UI TUI                     [Ctrl+C] │
├──────────┬─────────────────────────────────┤
│  Sidebar │  Main Content                    │
│           │                                  │
│  📊 Dash  │  (context-dependent)            │
│  📦 Inst  │                                  │
│  🔗 Subs  │                                  │
│  🎯 Probe │                                  │
│  ⚡ Speed  │                                  │
│  🛡 WARP   │                                  │
│  🔐 WG    │                                  │
│  📜 Cert  │                                  │
│           │                                  │
├──────────┴─────────────────────────────────┤
│  Status bar: Connected | Sing-box: running  │
└────────────────────────────────────────────┘
```

**Навигация:**

| Клавиша | Действие |
|---------|----------|
| `↑`/`↓` | Навигация по спискам / таблицам |
| `Enter` | Выбрать / открыть детали |
| `Tab` / `Shift+Tab` | Переключение между панелями |
| `Esc` / `q` | Назад / закрыть |
| `r` | Refresh (перезапросить данные) |
| `Ctrl+C` | Выход |

### 4.4 Экраны (views)

#### Dashboard (главная)

- Health check OK/FAIL
- Sing-box статус (running/stopped/error)
- CPU/память контейнеров (если доступно)
- Speed test status
- Prober status

#### Instances (список экземпляров)

- Таблица: имя, статус, порт, uptime
- Действия: Run/Stop/Check/Logs для каждого
- При выборе — детальный просмотр: конфиг, логи (tail), статус

#### Subscriptions

- Список подписок: имя, URL, кол-во нод, статус
- Добавление новой подписки (форма ввода)
- Refresh All
- При выборе — список нод подписки с результатами probing

#### Prober

- Статус: running/stopped
- Start/Stop кнопки
- Таблица результатов: нода, latency, status
- Sync из подписок

#### Speed Test

- Start/Stop
- Прогресс-бар текущего теста
- Результаты: download, upload, latency

#### WARP

- Статус аккаунта (device ID, license)
- Register / License bind
- Scan endpoints
- Результаты сканирования

#### WireGuard

- Generate keys
- Public key derivation
- Client config management
- Public IP check

#### Certificate

- Текущий сертификат (инфо, expiry)
- Generate self-signed
- Upload files
- Reality keypair

### 4.5 State Management

```rust
// app.rs (core state)
pub struct App {
    // Подключение к API
    pub client: ApiClient,

    // Навигация
    pub current_screen: Screen,
    pub sidebar_state: ListState,

    // Данные (обновляются асинхронно)
    pub health: Option<HealthResult>,
    pub status: Option<StatusResponse>,
    pub instances: Vec<NamedInstanceResponse>,
    pub subscriptions: Vec<SubscriptionResponse>,
    pub prober_status: Option<ProberStatus>,
    pub speedtest: Option<SpeedTestState>,
    pub warp_account: Option<WarpAccount>,

    // Polling-состояния
    pub loading: HashSet<LoadingKey>,
    pub error: Option<String>,
}

pub enum Screen {
    Dashboard,
    Instances,
    InstanceDetail(String),
    Subscriptions,
    SubscriptionDetail(String),
    Prober,
    SpeedTest,
    Warp,
    WireGuard,
    Certificate,
}
```

```rust
// event.rs (event loop)
pub enum Event {
    Tick,              // 250ms refresh
    Key(KeyEvent),     // Keyboard input
    DataLoaded(DataKey, Box<dyn Any>),
    Error(String),
}
```

### 4.6 Асинхронный polling

- Dashboard: health + status каждые 5 секунд
- Speedtest: состояние каждую секунду во время теста
- Prober: результаты каждые 10 секунд (если active)
- Logs: tail с debounce 500ms
- Всё остальное: только по запросу (enter, refresh)

Использовать `tokio::select!` в event loop:

```rust
loop {
    tokio::select! {
        // Keyboard input
        event = read_key() => handle_key(event),
        // Periodic refresh
        _ = sleep(Duration::from_secs(5)) => {
            if screen_visible(Screen::Dashboard) {
                tokio::spawn(refresh_health(client.clone()));
            }
        }
        // ... more channels
    }
}
```

### 4.7 Обработка ошибок

Все API-вызовы возвращают `anyhow::Result<T>`. Ошибки отображаются в статус-баре
и в отдельном popup (красный текст, `Enter` для закрытия).

```rust
// api.rs
pub enum ApiError {
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),
    #[error("API error: {status} - {message}")]
    Api { status: StatusCode, message: String },
    #[error("Connection refused: {0}")]
    Connection(String),
}
```

---

## 5. Prompt для AI (готовый к использованию)

Этот блок можно скопировать и отправить в чат ИИ целиком.

---

```
# Задача: Создать Rust TUI клиент для SingBox UI API

## API спецификация

Swagger 2.0 спецификация — 54 REST endpoints, 38 типов данных.
Все эндпоинты на `http://localhost:8080/api/`, без аутентификации.

## Что нужно сделать

1. Создать Rust-проект `singbox-tui` с:
   - TUI на `ratatui` + `crossterm`
   - HTTP-клиент на `reqwest` + `serde`
   - Асинхронность на `tokio`
   - CLI на `clap` (флаг `--url http://localhost:8080)

2. Структура:

### API Layer (вручную, ~800 строк)

Описание всех эндпоинтов и типов:

**System:**
- GET /health → 200 OK / 500 ErrorResponse

**SingBox (15 endpoints):**
- GET /singbox/status → {running: bool, config_ok: bool} (StatusResponse)
- GET /singbox/version → {version: string} (VersionResponse)
- POST /singbox/run → {message: string} (MessageResponse)
- POST /singbox/stop → {message: string} (MessageResponse)
- GET /singbox/config → sing-box config JSON (raw)
- POST /singbox/config → saves config (body: raw JSON)
- GET /singbox/logs → array of log lines
- GET /singbox/instances → [{name, config_path, created_at}] (array of NamedInstanceResponse)
- POST /singbox/instances/{name}/run → {}
- POST /singbox/instances/{name}/stop → {}
- GET /singbox/instances/{name}/status → {running: bool} (StatusResponse)
- GET /singbox/instances/{name}/config → config JSON (raw)
- POST /singbox/instances/{name}/config → saves config
- POST /singbox/instances/{name}/check → CheckConfigResponse
- DELETE /singbox/instances/{name} → {}
- GET /singbox/instances/{name}/logs → logs
- GET /singbox/containers → []
- POST /singbox/ensure-image → {}
- GET /singbox/certificate → CertificateInfo

**Certificate (5 endpoints):**
- GET /singbox/certificate → {cert_file, key_file, not_after, ...} (CertificateInfo)
- POST /singbox/certificate → body: {host, valid_for_years} (GenerateCertRequest) → {}
- POST /singbox/certificate/upload → multipart: cert + key files → {}
- POST /singbox/reality/keypair → {private_key, public_key} (RealityKeypairResponse)
- POST /singbox/reality/public-key → body: {private_key} → {public_key}
- POST /singbox/reality/check-tls → body: {domain, port} (CheckTLS13Request) → CheckTLS13Response

**Subscription (7 endpoints):**
- GET /subscription → list of subscriptions
- POST /subscription → body: {name, url, user_agent} (AddSubscriptionRequest) → {}
- DELETE /subscription/{id} → {}
- PATCH /subscription/{id}/settings → body: {user_agent, ...} (UpdateSettingsRequest) → {}
- POST /subscription/{id}/refresh → {}
- POST /subscription/refresh-all → {}
- GET /subscription/nodes → proxy nodes
- GET /subscription/user-agents → [string]

**Prober (12 endpoints):**
- GET /prober/status → ProberStatus {running: bool, ...}
- POST /prober/start → {}
- POST /prober/stop → {}
- GET /prober/best → best node result
- GET /prober/online → online nodes
- GET /prober/results → all results
- GET /prober/results/{tag} → result by tag
- PUT /prober/nodes → update nodes (body: ProberNodesRequest)
- POST /prober/nodes → add node (body: ProberNodeRequest)
- DELETE /prober/nodes → clear all
- DELETE /prober/nodes/{tag} → remove by tag
- POST /prober/save → save results
- POST /prober/sync → sync from subscription

**SpeedTest (3 endpoints):**
- POST /speedtest/start → {}
- POST /speedtest/stop → {}
- GET /speedtest/status → SpeedTestState {running, progress, download, upload, ...}

**WARP (5 endpoints):**
- GET /warp/account → WarpAccount {account_id, device_id, ...}
- DELETE /warp/account → {}
- POST /warp/register → WarpRegisterResponse {config, ...}
- POST /warp/license → body: {license_key} (LicenseBindRequest) → {}
- POST /warp/scan → body: WarpScanConfig → [WarpEndpointResult]

**WireGuard (9 endpoints):**
- GET /wireguard/client-config → ClientConfigFile
- POST /wireguard/client-config → save config
- GET /wireguard/client-files → [ClientConfigFile]
- POST /wireguard/save-client-file → body: {name, content} (SaveClientFileRequest) → {}
- GET /wireguard/keys-cache → keys info
- POST /wireguard/keygen → body: {wg_conf_path} (WireGuardKeyRequest) → WireGuardKeyPair
- POST /wireguard/pubkey → body: {private_key} (DerivePublicKeyRequest) → key string
- GET /wireguard/public-ip → {ip: string} (PublicIPResponse)

### TUI Screens

Создать 8 экранов с навигацией через боковое меню:

1. **Dashboard** — health check + sing-box статус + быстрые действия (run/stop)
2. **Instances** — список Docker-контейнеров, управление (run/stop/logs/config)
3. **Subscriptions** — CRUD подписок, просмотр нод, refresh
4. **Prober** — статус, результаты, start/stop
5. **SpeedTest** — запуск, прогресс, результаты
6. **WARP** — регистрация, лицензия, сканирование
7. **WireGuard** — генерация ключей, управление клиентскими конфигами
8. **Certificate** — просмотр сертификата, генерация, Reality

### Технические требования

- TUI должен работать в терминалах 80x24 и больше
- Все HTTP-запросы асинхронные (не блокируют UI)
- Periodic polling для dashboard (health/status каждые 5с)
- Обработка ошибок: таймауты, connection refused — показывать в статус-баре
- Конфигурация: аргумент `--url` (по умолчанию http://localhost:8080)
- Ctrl+C для выхода, q/Esc для назад

### Cargo.toml

```toml
[package]
name = "singbox-tui"
version = "0.1.0"
edition = "2024"

[dependencies]
ratatui = "0.29"
crossterm = "0.28"
reqwest = { version = "0.12", features = ["json"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tokio = { version = "1", features = ["full"] }
clap = { version = "4", features = ["derive"] }
anyhow = "1"
thiserror = "2"
tracing = "0.1"
tracing-subscriber = "0.3"
```

### Структура файлов

```
singbox-tui/
├── Cargo.toml
├── src/
│   ├── main.rs           # CLI + запуск
│   ├── app.rs            # Состояние + event loop
│   ├── api.rs            # HTTP-клиент (reqwest)
│   ├── types.rs          # Все типы (serde)
│   ├── ui/
│   │   ├── mod.rs        # Layout root
│   │   ├── dashboard.rs
│   │   ├── instances.rs
│   │   ├── subscriptions.rs
│   │   ├── prober.rs
│   │   ├── speedtest.rs
│   │   ├── warp.rs
│   │   ├── wireguard.rs
│   │   ├── certificate.rs
│   │   └── common.rs     # Spinner, table helpers, form input
│   └── event.rs          # Keyboard events + polling
```

### Важные детали типов (из Swagger)

```rust
// === System ===
#[derive(Debug, Deserialize, Serialize)]
pub struct ErrorResponse {
    pub error: String,
}

// === SingBox ===
#[derive(Debug, Deserialize, Serialize)]
pub struct StatusResponse {
    pub running: Option<bool>,
    pub config_ok: Option<bool>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct VersionResponse {
    pub version: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct NamedInstanceResponse {
    pub name: String,
    pub config_path: Option<String>,
    pub created_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct CheckConfigResponse {
    pub valid: Option<bool>,
    pub errors: Option<Vec<String>>,
}

// === Certificate ===
#[derive(Debug, Deserialize, Serialize)]
pub struct CertificateInfo {
    pub cert_file: Option<String>,
    pub key_file: Option<String>,
    pub not_after: Option<String>,
    pub issuer: Option<String>,
    pub subject: Option<String>,
    pub sans: Option<Vec<String>>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct RealityKeypairResponse {
    pub private_key: String,
    pub public_key: String,
}

// === Prober ===
#[derive(Debug, Deserialize, Serialize)]
pub struct ProberStatus {
    pub running: bool,
    pub nodes_total: Option<i32>,
    pub nodes_online: Option<i32>,
    pub last_probe: Option<String>,
    pub interval: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ProberNodeRequest {
    pub tag: String,
    pub host: String,
    pub port: i32,
    pub method: String,
}

// === SpeedTest ===
#[derive(Debug, Deserialize, Serialize)]
pub struct SpeedTestState {
    pub running: Option<bool>,
    pub progress: Option<f64>,
    pub download_speed: Option<f64>,
    pub upload_speed: Option<f64>,
    pub latency: Option<f64>,
    pub server: Option<String>,
    pub error: Option<String>,
}

// === WARP ===
#[derive(Debug, Deserialize, Serialize)]
pub struct WarpAccount {
    pub account_id: Option<String>,
    pub device_id: Option<String>,
    pub device_name: Option<String>,
    pub license: Option<String>,
    pub account_type: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct WarpRegisterResponse {
    pub account_id: Option<String>,
    pub device_id: Option<String>,
    pub device_name: Option<String>,
    pub config: Option<String>,
    pub account_type: Option<String>,
}

// === WireGuard ===
#[derive(Debug, Deserialize, Serialize)]
pub struct WireGuardKeyPair {
    pub private_key: String,
    pub public_key: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ClientConfigFile {
    pub name: Option<String>,
    pub content: Option<String>,
}
```

Напиши полностью рабочий проект. Все эндпоинты должны быть реализованы.
TUI должен быть навигабельным и функциональным. Используй ratatui с блоками,
таблицами, прогресс-барами и формами ввода.
```

---

## 6. Порядок действий для разработчика

1. **Сгенерировать или написать API-клиент** (вариант A/B/C из раздела 3)
2. **Создать проект** `cargo new singbox-tui`
3. **Добавить зависимости** из Cargo.toml выше
4. **Написать `types.rs`** — все 38 структур с `#[derive(Debug, Deserialize, Serialize)]`
5. **Написать `api.rs`** — HTTP-клиент со всеми методами
6. **Написать `event.rs`** — event loop, keyboard, polling
7. **Написать `app.rs`** — состояние, переходы, загрузка данных
8. **Написать все `ui/` модули** — экраны и компоненты
9. **Собрать и протестировать** с живым бэкендом
10. **Проверить крайние случаи:** таймауты, connection refused, resize терминала

## 7. Требования к качеству

- TUI корректно обрабатывает `resize` терминала
- Все запросы с таймаутом (default: 10s)
- Graceful shutdown при Ctrl+C
- Ошибки API не роняют приложение — показываются в статус-баре
- Loading spinner на время запросов
- Таблицы с сортировкой (по умолчанию — по имени/статусу)

## 8. Out of Scope

- Аутентификация (в бэкенде её нет)
- Поддержка нескольких серверов одновременно
- Редактирование конфигов в TUI (только просмотр и сохранение как есть)
- Графики (CPU/RAM) — только текстовые значения
- i18n — только английский язык интерфейса
