# TUI Prototype for singbox-ui (yazi-style)

## Описание

Rust TUI для управления sing-box backend через REST API. Стиль навигации — как в yazi
(файловый менеджер, две панели, vim-like клавиши, дерево сущностей).

## Стек

| Компонент | Крейт |
|---|---|
| TUI | `ratatui` + `crossterm` |
| HTTP | `reqwest` (async) |
| Runtime | `tokio` |
| JSON | `serde` + `serde_json` |
| Редактор | `tui-textarea` |
| Время | `chrono` |

## Архитектура: вертикальные срезы

```
tui/
├── Cargo.toml
├── src/
│   ├── main.rs
│   │
│   ├── core/                  # общая инфраструктура
│   │   ├── mod.rs
│   │   ├── app.rs             # App — главная структура состояния
│   │   ├── entity.rs          # Entity — общий трейт для навигации
│   │   ├── tree.rs            # навигационное дерево с курсором
│   │   ├── layout.rs          # split-panel + статус-бар + заголовок
│   │   ├── keybind.rs         # глобальные клавиши + диспетчеризация
│   │   ├── input.rs           # текстовый инпут/диалог
│   │   ├── editor.rs          # полноэкранный JSON редактор
│   │   └── api_client.rs      # базовый HTTP-клиент
│   │
│   ├── slices/                # каждый домен — вертикальный срез
│   │   ├── mod.rs
│   │   ├── subscription/      # подписки
│   │   ├── prober/            # пробер
│   │   ├── singbox/           # sing-box конфиги/контейнеры
│   │   ├── speedtest/         # скорость
│   │   ├── wireguard/         # WireGuard
│   │   ├── warp/              # Cloudflare WARP
│   │   └── certificate/       # TLS сертификаты
│   │
│   └── app.rs                 # сборка core + slices
```

### core/entity.rs — базовый трейт

```rust
pub trait Entity {
    fn id(&self) -> &str;
    fn label(&self) -> &str;
    fn icon(&self) -> &str;              // "~", ">", "●", "○"
    fn kind(&self) -> EntityKind;
    fn can_have_children(&self) -> bool;
    fn children(&self) -> &[Box<dyn Entity>];
    fn commands(&self) -> Vec<Command>;
    fn on_action(&mut self, action: &Action, app: &mut App);
}

pub enum EntityKind {
    Root,
    Section(SectionKind),   // Subscriptions, Prober, Singbox, WireGuard, WARP, Certificate
    Subscription,
    Node,
    Config,
    Logs,
    Status,
    Keys,
    Account,
    Cert,
}

pub struct Command {
    pub key: char,
    pub label: &'static str,
    pub action: Action,
}

pub enum Action {
    Refresh,
    Delete,
    Add,
    Run,
    Stop,
    Start,
    Probe,
    Speedtest,
    EditConfig,
    ViewLogs,
    GenerateKeys,
    Register,
    BindLicense,
    Scan,
    Sync,
    Custom(String),
}
```

### core/app.rs — состояние

```rust
pub struct App {
    pub tree: Box<dyn Entity>,              // корень дерева
    pub cursor: Vec<usize>,                 // путь к выбранной сущности
    pub mode: AppMode,                      // Normal | Input | Editor | Confirm
    pub loading: bool,
    pub status_message: Option<String>,
    pub error_message: Option<String>,
    pub editor_state: Option<EditorState>,
    pub input_state: Option<InputState>,
    pub confirm_state: Option<ConfirmState>,
}
```

### core/layout.rs — разбивка экрана

```
┌─── singbox-ui ────────────────────── [?] ─┐
│ ┌──────────┬──────────────────────────────┐│
│ │ ~ Subscr │  <view зависит от EntityKind> ││
│ │   > my   │                              ││
│ │   > work │                              ││
│ │ ~ Prober │                              ││
│ │ ~ Singbx │                              ││
│ │ ~ WireGu │                              ││
│ │ ~ WARP   │                              ││
│ ├──────────┴──────────────────────────────┤│
│ │ Status bar / help                       ││
│ └─────────────────────────────────────────┘│
└────────────────────────────────────────────┘
```

- Левая панель: 30% ширины, рендерит дерево
- Правая панель: 70% ширины, рендерит view по EntityKind
- Нижняя панель: 1 строка, контекстные комманды + статус

## Срез: subscription

### entity.rs

```
EntitySubscription (section, can_have_children=true)
  └── EntitySubItem (одна подписка, can_have_children=true)
        └── EntityNode (прокси-нода, can_have_children=false)
```

### api.rs

- `list()` -> `GET /api/subscription` — список подписок с нодами
- `add(name, url, user_agent)` -> `POST /api/subscription` — добавить подписку
- `refresh(id)` -> `POST /api/subscription/{id}/refresh` — обновить одну
- `refresh_all()` -> `POST /api/subscription/refresh-all` — обновить все
- `delete(id)` -> `DELETE /api/subscription/{id}` — удалить
- `update_settings(id, auto_update, interval)` -> `PATCH /api/subscription/{id}/settings`
- `user_agents()` -> `GET /api/subscription/user-agents` — список UA

### view.rs

- **Section view**: таблица подписок (Name | Nodes | AutoUpdate | LastUpdated)
- **SubItem view**: карточка (URL, UA, LastUpdated) + таблица нод (Tag | Protocol | Address | Latency | Online)
- **Node view**: детали ноды (протокол, адрес:порт, latency, success rate, speed)

### commands.rs

- `a` — Add (инпут: name, url, user-agent)
- `r` — Refresh (текущую или все)
- `d` — Delete (с подтверждением)
- `s` — Edit settings (auto-update, interval)
- `p` — Probe now (на ноде)
- `t` — Speedtest (на ноде)

## Срез: prober

### entity.rs

```
EntityProber (section, can_have_children=true)
  ├── EntityProberStatus (может быть дочерним как "Status")
  └── EntityNode (прокси-нода в пробере, can_have_children=false)
```

### api.rs

- `status()` -> `GET /api/prober/status`
- `results()` -> `GET /api/prober/results`
- `best()` -> `GET /api/prober/best`
- `online()` -> `GET /api/prober/online`
- `add_node(...)` -> `POST /api/prober/nodes`
- `remove_node(tag)` -> `DELETE /api/prober/nodes/{tag}`
- `start()` -> `POST /api/prober/start`
- `stop()` -> `POST /api/prober/stop`
- `sync()` -> `POST /api/prober/sync`
- `save()` -> `POST /api/prober/save`

### view.rs

- **Section view**: статус (running/stopped), кол-во нод, best node
- **Node view**: latency, failCount, successRate, status

### commands.rs

- `s` — Start/Stop prober
- `y` — Sync from subscriptions
- `r` — Refresh
- `d` — Remove node (на ноде)

## Срез: singbox

### entity.rs

```
EntitySingbox (section, can_have_children=true)
  └── EntityConfig (один именованный конфиг, can_have_children=true)
        ├── EntityConfigContent ("Config", отображает JSON, можно редактировать)
        └── EntityLogs ("Logs", отображает логи контейнера)
```

### api.rs

- `version()` -> `GET /api/singbox/version`
- `config()` -> `GET /api/singbox/config`
- `save_config(body)` -> `POST /api/singbox/config`
- `run()` -> `POST /api/singbox/run`
- `stop()` -> `POST /api/singbox/stop`
- `logs()` -> `GET /api/singbox/logs`
- `status()` -> `GET /api/singbox/status`
- `ensure_image()` -> `POST /api/singbox/ensure-image`
- `list_instances()` -> `GET /api/singbox/instances`
- `load_named_config(name)` -> `GET /api/singbox/instances/{name}/config`
- `save_named_config(name, body)` -> `POST /api/singbox/instances/{name}/config`
- `run_named(name)` -> `POST /api/singbox/instances/{name}/run`
- `stop_named(name)` -> `POST /api/singbox/instances/{name}/stop`
- `named_status(name)` -> `GET /api/singbox/instances/{name}/status`
- `named_logs(name)` -> `GET /api/singbox/instances/{name}/logs`
- `delete_named(name)` -> `DELETE /api/singbox/instances/{name}`
- `list_containers()` -> `GET /api/singbox/containers`

### view.rs

- **Section view**: версия sing-box, список конфигов (Name | Running)
- **Config view**: статус, container ID, команды
- **ConfigContent view**: JSON-текст (с подсветкой)
- **Logs view**: скроллируемый текст логов

### commands.rs

- `e` — Edit config (полноэкранный JSON редактор)
- `r` — Run named container
- `s` — Stop named container
- `l` — View logs
- `d` — Delete config (с подтверждением)
- `a` — Create new config (имя + JSON)

## Остальные срезы (заглушки)

### speedtest

- `POST /api/speedtest/start`, `GET /api/speedtest/status`, `POST /api/speedtest/stop`
- Команды: `s` — Start/Stop

### wireguard

- `POST /api/wireguard/keygen`, `GET /api/wireguard/client-config`, `GET /api/wireguard/client-files`
- Команды: `g` — Generate keys, `c` — Show client config
- Вид: ключи, IP, список .conf файлов

### warp

- `GET /api/warp/account`, `POST /api/warp/register`, `POST /api/warp/license`, `POST /api/warp/scan`
- Команды: `n` — Register, `l` — Bind license, `s` — Scan endpoints
- Вид: статус аккаунта, лицензия

### certificate

- `GET /api/singbox/certificate`, `POST /api/singbox/certificate`
- Команды: `g` — Generate cert, `u` — Upload
- Вид: информация о сертификате (domain, valid until, fingerprint)

## Навигация

### Клавиши (глобальные)

| Клавиша | Действие |
|---|---|
| `↑/↓`, `j/k` | Навигация по дереву |
| `→`, `l`, `Enter` | Раскрыть / войти |
| `←`, `h` | Свернуть / назад |
| `g` | В начало дерева |
| `G` | В конец дерева |
| `q` | Выход |
| `?` | Help overlay |
| `Space` | Выполнить команду на текущей сущности |

### Клавиши (команды)

Команды привязаны к Entity — отображаются в статус-баре, выполняются по нажатию:

- `a`dd, `r`efresh, `d`elete, `s`tart/stop, `e`dit, `p`robe, `t`est, `y` sync, `g`enerate, `l`ogs, `n` register

### Модальные режимы

- **Normal** — навигация, две панели
- **Input** — текстовый инпут (Esc — отмена, Enter — подтвердить)
- **Editor** — полноэкранный JSON редактор (Esc — отмена, Ctrl+S — сохранить)
- **Confirm** — подтверждение действия (y/n)

## ROOT entity (сборка)

Корневое Entity собирается из всех срезов при старте:

```rust
fn build_root(slices: &[Box<dyn Slice>]) -> Box<dyn Entity> {
    let mut root = EntityRoot::new();
    for slice in slices {
        root.children.push(slice.root_entity());
    }
    Box::new(root)
}
```

Каждый срез реализует трейт `Slice`:

```rust
pub trait Slice {
    fn name(&self) -> &'static str;
    fn root_entity(&self) -> Box<dyn Entity>;
    fn on_action(&mut self, action: &Action, app: &mut App);
}
```

## Прототип: scope

В прототип входят:

1. **core/** — app, entity, tree, layout, keybind, api_client
2. **slices/subscription/** — полный срез (list, add, refresh, delete, node details)
3. **slices/prober/** — полный срез (status, results, start/stop, sync)
4. **slices/singbox/** — полный срез (configs, run/stop, logs, JSON editor)
5. **slices/speedtest/** — базовая заглушка
6. **slices/wireguard/** — базовая заглушка
7. **slices/warp/** — базовая заглушка
8. **slices/certificate/** — базовая заглушка

Каждая заглушка показывает: секцию в дереве, статический view, команды-заглушки с сообщением "Not implemented".

## Будущие улучшения (за рамками прототипа)

- Сохранение состояния между запусками
- Поддержка нескольких профилей/серверов
- Темы оформления
- История команд
- Режим реального времени (логи, пробинг)
- Аутентификация к API
