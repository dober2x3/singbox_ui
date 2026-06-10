# TUI Prototype Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working Rust TUI prototype for singbox-ui backend in yazi style.

**Architecture:** Vertical slices — each backend domain is a self-contained module (entity, API client, view renderer, commands). Core layer provides tree navigation, split-panel layout, keybind dispatch, and base HTTP client.

**Tech Stack:** Rust, ratatui + crossterm (TUI), reqwest (async HTTP), tokio (async runtime), serde + serde_json (JSON), tui-textarea (editor), chrono (dates).

---

## File Structure

```
singbox_ui/
└── tui/
    ├── Cargo.toml
    └── src/
        ├── main.rs
        ├── app.rs                          # App state, update(), event loop
        ├── core/
        │   ├── mod.rs
        │   ├── entity.rs                   # Entity trait, EntityKind, Command, Action
        │   ├── tree.rs                     # Tree navigation, cursor, expand/collapse
        │   ├── layout.rs                   # Split-panel + status bar rendering
        │   ├── keybind.rs                  # Key event → Command dispatch
        │   ├── input.rs                    # Text input overlay
        │   ├── editor.rs                   # Fullscreen JSON editor
        │   └── api_client.rs               # Base HTTP client (reqwest wrapper)
        └── slices/
            ├── mod.rs                      # Slice trait + registry
            ├── subscription.rs             # Full: subscriptions + nodes
            ├── prober.rs                   # Full: prober status + results
            ├── singbox.rs                  # Full: configs + containers + logs + JSON editor
            ├── speedtest.rs                # Stub
            ├── wireguard.rs                # Stub
            ├── warp.rs                     # Stub
            └── certificate.rs              # Stub
```

---

### Task 1: Project scaffold

**Files:**
- Create: `tui/Cargo.toml`
- Create: `tui/src/main.rs` (minimal)

- [ ] **Step 1: Create Cargo.toml**

```toml
[package]
name = "singbox-tui"
version = "0.1.0"
edition = "2024"

[dependencies]
ratatui = "0.29"
crossterm = "0.28"
tokio = { version = "1", features = ["full"] }
reqwest = { version = "0.12", features = ["json"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tui-textarea = "0.7"
chrono = { version = "0.4", features = ["serde"] }
anyhow = "1"
thiserror = "1"
```

- [ ] **Step 2: Create minimal main.rs**

```rust
fn main() -> anyhow::Result<()> {
    println!("singbox-tui prototype");
    Ok(())
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd tui && cargo check`
Expected: cargo downloads deps, compiles successfully

- [ ] **Step 4: Commit**

```bash
cd tui && git add Cargo.toml src/main.rs
git commit -m "feat(tui): scaffold Rust project with ratatui + reqwest"
```

---

### Task 2: Core types — Entity, Command, Action, EntityKind

**Files:**
- Create: `tui/src/core/mod.rs`
- Create: `tui/src/core/entity.rs`

- [ ] **Step 1: Write test for Entity trait + EntityKind**

```rust
// tui/src/core/entity.rs — tests module at bottom

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_entity_creation() {
        let e = EntityNode::new(
            "n1".into(),
            "JP-Tokyo".into(),
            "●".into(),
            EntityKind::Node,
            false,
            vec![],
            vec![Command::new('p', "Probe", Action::Probe)],
        );
        assert_eq!(e.id(), "n1");
        assert_eq!(e.label(), "JP-Tokyo");
        assert_eq!(e.icon(), "●");
        assert_eq!(e.can_have_children(), false);
        assert_eq!(e.commands().len(), 1);
        assert_eq!(e.commands()[0].key, 'p');
    }

    #[test]
    fn test_entity_children() {
        let child = EntityNode::new("c1".into(), "Child".into(), "●".into(), EntityKind::Node, false, vec![], vec![]);
        let parent = EntityNode::new("p1".into(), "Parent".into(), ">".into(), EntityKind::Subscription, true, vec![child.clone()], vec![]);
        assert_eq!(parent.can_have_children(), true);
        assert_eq!(parent.children().len(), 1);
        assert_eq!(parent.children()[0].id(), "c1");
    }

    #[test]
    fn test_section_kind() {
        let section = EntityNode::new("sub".into(), "Subscriptions".into(), "~".into(), EntityKind::Section(SectionKind::Subscriptions), true, vec![], vec![]);
        assert!(matches!(section.kind(), EntityKind::Section(SectionKind::Subscriptions)));
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tui && cargo test`
Expected: compile error — EntityNode and friends don't exist yet

- [ ] **Step 3: Write entity.rs implementation**

```rust
use std::fmt;

/// Action that can be performed on an Entity.
#[derive(Debug, Clone, PartialEq)]
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

/// A command bound to a key, shown in status bar.
#[derive(Debug, Clone)]
pub struct Command {
    pub key: char,
    pub label: &'static str,
    pub action: Action,
}

impl Command {
    pub fn new(key: char, label: &'static str, action: Action) -> Self {
        Self { key, label, action }
    }
}

/// Top-level section kinds.
#[derive(Debug, Clone, PartialEq)]
pub enum SectionKind {
    Subscriptions,
    Prober,
    Singbox,
    Speedtest,
    WireGuard,
    WARP,
    Certificate,
}

impl fmt::Display for SectionKind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            SectionKind::Subscriptions => write!(f, "Subscriptions"),
            SectionKind::Prober => write!(f, "Prober"),
            SectionKind::Singbox => write!(f, "Singbox"),
            SectionKind::Speedtest => write!(f, "Speedtest"),
            SectionKind::WireGuard => write!(f, "WireGuard"),
            SectionKind::WARP => write!(f, "WARP"),
            SectionKind::Certificate => write!(f, "Certificate"),
        }
    }
}

/// Entity kind determines how the entity is rendered in the right panel.
#[derive(Debug, Clone, PartialEq)]
pub enum EntityKind {
    Root,
    Section(SectionKind),
    Subscription,
    Node,
    Config,
    Logs,
    Status,
    Keys,
    Account,
    Cert,
}

/// Core trait for navigation tree entities.
pub trait Entity {
    fn id(&self) -> &str;
    fn label(&self) -> &str;
    fn icon(&self) -> &str;
    fn kind(&self) -> &EntityKind;
    fn can_have_children(&self) -> bool;
    fn children(&self) -> &[Box<dyn Entity>];
    fn commands(&self) -> &[Command];
}

/// Concrete entity node implementing the Entity trait.
#[derive(Debug, Clone)]
pub struct EntityNode {
    id: String,
    label: String,
    icon: String,
    kind: EntityKind,
    can_have_children: bool,
    children: Vec<Box<dyn Entity>>,
    commands: Vec<Command>,
}

impl EntityNode {
    pub fn new(
        id: String,
        label: String,
        icon: String,
        kind: EntityKind,
        can_have_children: bool,
        children: Vec<Box<dyn Entity>>,
        commands: Vec<Command>,
    ) -> Self {
        Self { id, label, icon, kind, can_have_children, children, commands }
    }
}

impl Entity for EntityNode {
    fn id(&self) -> &str { &self.id }
    fn label(&self) -> &str { &self.label }
    fn icon(&self) -> &str { &self.icon }
    fn kind(&self) -> &EntityKind { &self.kind }
    fn can_have_children(&self) -> bool { self.can_have_children }
    fn children(&self) -> &[Box<dyn Entity>] { &self.children }
    fn commands(&self) -> &[Command] { &self.commands }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd tui && cargo test`
Expected: 3 tests pass

- [ ] **Step 5: Commit**

```bash
cd tui
git add src/core/mod.rs src/core/entity.rs
git commit -m "feat(tui): add core Entity trait, EntityKind, Command, Action"
```

---

### Task 3: Tree navigation

**Files:**
- Create: `tui/src/core/tree.rs`

- [ ] **Step 1: Write test for Tree navigation**

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::entity::*;

    fn make_test_root() -> Box<EntityNode> {
        let n1 = Box::new(EntityNode::new("n1".into(), "JP-01".into(), "●".into(), EntityKind::Node, false, vec![], vec![]));
        let n2 = Box::new(EntityNode::new("n2".into(), "SG-02".into(), "●".into(), EntityKind::Node, false, vec![], vec![]));
        let sub = Box::new(EntityNode::new("sub".into(), "my-vpn".into(), ">".into(), EntityKind::Subscription, true, vec![n1, n2], vec![]));
        let section = EntityNode::new("sec".into(), "Subscriptions".into(), "~".into(), EntityKind::Section(SectionKind::Subscriptions), true, vec![sub], vec![]);
        section
    }

    #[test]
    fn test_flat_visible_list_root() {
        let root = make_test_root();
        let tree = Tree::new(root);
        let visible = tree.visible_items();
        // Root itself + "sec" with sub collapsed = 2 items
        assert_eq!(visible.len(), 2);
        assert_eq!(visible[1].id(), "sec");
    }

    #[test]
    fn test_expand_reveals_children() {
        let root = make_test_root();
        let mut tree = Tree::new(root);
        // Initially "sub" is collapsed, so we see: root, sec, sub
        // Sections start collapsed. So visible = [root, sec]
        // Expand sec -> visible = [root, sec, sub]
        // Expand sub -> visible = [root, sec, sub, n1, n2]
        tree.toggle_expand(); // expand "sec"
        assert_eq!(tree.visible_items().len(), 3);
        assert_eq!(tree.visible_items()[2].id(), "sub");

        tree.move_down(); // select "sub"
        tree.toggle_expand(); // expand "sub"
        assert_eq!(tree.visible_items().len(), 5);
    }

    #[test]
    fn test_cursor_movement() {
        let root = make_test_root();
        let mut tree = Tree::new(root);
        assert_eq!(tree.cursor(), 0); // Root selected

        tree.move_down(); // sec
        assert_eq!(tree.cursor(), 1);
        assert_eq!(tree.selected().id(), "sec");
    }

    #[test]
    fn test_cursor_bounds() {
        let root = make_test_root();
        let mut tree = Tree::new(root);
        // root + sec = 2 items
        tree.move_down(); // sec
        tree.move_down(); // should stay at sec (bottom)
        assert_eq!(tree.cursor(), 1);
    }

    #[test]
    fn test_parent_navigation() {
        let root = make_test_root();
        let mut tree = Tree::new(root);
        tree.toggle_expand(); // expand sec
        tree.move_down(); // sub
        tree.toggle_expand(); // expand sub
        tree.move_down(); // n1
        tree.move_down(); // n2
        // Go to parent -> should collapse n2 level... 
        // Navigate to parent entity (go up one hierarchy level)
        tree.navigate_to_parent();
        assert_eq!(tree.selected().id(), "sub");
    }
}
```

- [ ] **Step 2: Write tree.rs implementation**

```rust
use crate::core::entity::{Entity, EntityKind, EntityNode};

/// Wraps a root Entity and manages visible-item navigation.
pub struct Tree {
    root: Box<EntityNode>,
    collapsed: std::collections::HashSet<String>,
    cursor: usize,
    visible: Vec<String>, // IDs of visible items in display order
}

impl Tree {
    pub fn new(root: Box<EntityNode>) -> Self {
        let collapsed = std::collections::HashSet::new();
        // All sections start collapsed
        let mut tree = Self { root, collapsed, cursor: 0, visible: vec![] };
        tree.rebuild_visible();
        tree
    }

    /// Rebuild the flat visible list from the tree hierarchy.
    fn rebuild_visible(&mut self) {
        self.visible.clear();
        self.flatten(&self.root, 0);
        if self.cursor >= self.visible.len() {
            self.cursor = if self.visible.is_empty() { 0 } else { self.visible.len() - 1 };
        }
    }

    /// Recursively flatten visible nodes.
    fn flatten(&self, node: &EntityNode, _depth: usize) {
        self.visible.push(node.id().to_string());
        if self.collapsed.contains(node.id()) {
            return;
        }
        for child in &node.children {
            // We need to downcast; for simplicity in prototype, EntityNode children are Box<dyn Entity>
            // but we store Box<EntityNode>. Let's adjust EntityNode to store children as Vec<Box<EntityNode>>.
        }
    }

    pub fn cursor(&self) -> usize { self.cursor }
    pub fn visible_items(&self) -> &[String] { &self.visible }

    pub fn selected(&self) -> &EntityNode {
        self.find_by_id(&self.visible[self.cursor])
    }

    fn find_by_id(&self, id: &str) -> &EntityNode {
        self.find_recursive(&self.root, id).unwrap()
    }

    fn find_recursive<'a>(&'a self, node: &'a EntityNode, id: &str) -> Option<&'a EntityNode> {
        if node.id() == id { return Some(node); }
        for child in &node.children {
            if let Some(found) = self.find_recursive(child, id) { return Some(found); }
        }
        None
    }

    pub fn move_down(&mut self) {
        if self.cursor + 1 < self.visible.len() {
            self.cursor += 1;
        }
    }

    pub fn move_up(&mut self) {
        if self.cursor > 0 {
            self.cursor -= 1;
        }
    }

    pub fn toggle_expand(&mut self) {
        let id = self.visible[self.cursor].clone();
        if self.collapsed.contains(&id) {
            self.collapsed.remove(&id);
        } else {
            self.collapsed.insert(id);
        }
        self.rebuild_visible();
    }

    pub fn is_expanded(&self) -> bool {
        let id = &self.visible[self.cursor];
        !self.collapsed.contains(id)
    }

    pub fn navigate_to_parent(&mut self) {
        // Find the parent of the currently selected node
        let selected_id = &self.visible[self.cursor];
        if let Some(parent) = self.find_parent(&self.root, selected_id) {
            let parent_id = parent.id().to_string();
            // Focus on the parent
            if let Some(pos) = self.visible.iter().position(|id| id == &parent_id) {
                self.cursor = pos;
            }
        }
    }

    fn find_parent<'a>(&'a self, node: &'a EntityNode, child_id: &str) -> Option<&'a EntityNode> {
        for child in &node.children {
            if child.id() == child_id { return Some(node); }
            if let Some(found) = self.find_parent(child, child_id) { return Some(found); }
        }
        None
    }

    /// Navigate cursor to the first element.
    pub fn go_to_top(&mut self) { self.cursor = 0; }

    /// Navigate cursor to the last element.
    pub fn go_to_bottom(&mut self) {
        if !self.visible.is_empty() {
            self.cursor = self.visible.len() - 1;
        }
    }
}
```

Use concrete `Vec<Box<EntityNode>>` instead of `Vec<Box<dyn Entity>>` for simplicity (no trait objects needed for prototype).

- [ ] **Step 3: Use concrete EntityNode children**

Change entity.rs so EntityNode stores `Vec<Box<EntityNode>>` instead of `Vec<Box<dyn Entity>>`:

```rust
pub struct EntityNode {
    pub id: String,
    pub label: String,
    pub icon: String,
    pub kind: EntityKind,
    pub can_have_children: bool,
    pub children: Vec<Box<EntityNode>>,
    pub commands: Vec<Command>,
}
```

And update the Entity trait implementation accordingly. The trait can still work because we implement children() as returning a slice of trait objects:

```rust
impl Entity for EntityNode {
    fn children(&self) -> &[Box<dyn Entity>] {
        // This won't work directly with Box<EntityNode>
        // Let's drop the trait approach for the prototype and just use EntityNode directly
    }
}
```

- [ ] **Step 4: Refactor entity.rs to concrete EntityNode only (no trait)**

```rust
use std::fmt;

#[derive(Debug, Clone, PartialEq)]
pub enum Action { /* ... same as before ... */ }

#[derive(Debug, Clone)]
pub struct Command {
    pub key: char,
    pub label: &'static str,
    pub action: Action,
}

impl Command {
    pub fn new(key: char, label: &'static str, action: Action) -> Self {
        Self { key, label, action }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub enum SectionKind {
    Subscriptions, Prober, Singbox, Speedtest, WireGuard, WARP, Certificate,
}

#[derive(Debug, Clone, PartialEq)]
pub enum EntityKind {
    Root,
    Section(SectionKind),
    Subscription,
    Node,
    Config,
    Logs,
    Status,
    Keys,
    Account,
    Cert,
}

/// Concrete node in the navigation tree.
#[derive(Debug, Clone)]
pub struct EntityNode {
    pub id: String,
    pub label: String,
    pub icon: String,
    pub kind: EntityKind,
    pub can_have_children: bool,
    pub children: Vec<Box<EntityNode>>,
    pub commands: Vec<Command>,
}

impl EntityNode {
    pub fn new(
        id: String, label: String, icon: String, kind: EntityKind,
        can_have_children: bool, children: Vec<Box<EntityNode>>, commands: Vec<Command>,
    ) -> Self {
        Self { id, label, icon, kind, can_have_children, children, commands }
    }

    pub fn is_section(&self) -> bool {
        matches!(self.kind, EntityKind::Section(_) | EntityKind::Root)
    }
}
```

- [ ] **Step 5: Update Tree to use concrete EntityNode**

```rust
use crate::core::entity::EntityNode;
use std::collections::HashSet;

pub struct Tree {
    root: Box<EntityNode>,
    collapsed: HashSet<String>,
    cursor: usize,
    visible: Vec<String>,
}

impl Tree {
    pub fn new(root: Box<EntityNode>) -> Self {
        let collapsed = HashSet::new();
        let mut tree = Self { root, collapsed, cursor: 0, visible: vec![] };
        tree.rebuild_visible();
        tree
    }

    fn rebuild_visible(&mut self) {
        self.visible.clear();
        self.flatten(&self.root);
        if self.cursor >= self.visible.len() && !self.visible.is_empty() {
            self.cursor = self.visible.len() - 1;
        }
    }

    fn flatten(&self, node: &EntityNode) {
        self.visible.push(node.id.clone());
        if self.collapsed.contains(&node.id) { return; }
        for child in &node.children {
            self.flatten(child);
        }
    }

    pub fn cursor(&self) -> usize { self.cursor }
    pub fn visible_ids(&self) -> &[String] { &self.visible }
    pub fn visible_len(&self) -> usize { self.visible.len() }

    pub fn selected<'a>(&'a self) -> &'a EntityNode {
        self.find_by_id(&self.visible[self.cursor]).unwrap()
    }

    pub fn selected_mut<'a>(&'a mut self) -> &'a mut EntityNode {
        let id = self.visible[self.cursor].clone();
        self.find_mut(&mut self.root, &id).unwrap()
    }

    fn find_by_id<'a>(&'a self, id: &str) -> Option<&'a EntityNode> {
        self.find_rec(&self.root, id)
    }

    fn find_rec<'a>(&'a self, node: &'a EntityNode, id: &str) -> Option<&'a EntityNode> {
        if node.id == id { return Some(node); }
        for child in &node.children {
            if let Some(found) = self.find_rec(child, id) { return Some(found); }
        }
        None
    }

    fn find_mut<'a>(&'a self, node: &'a mut EntityNode, id: &str) -> Option<&'a mut EntityNode> {
        if node.id == id { return Some(node); }
        for child in &mut node.children {
            if let Some(found) = self.find_mut(child, id) { return Some(found); }
        }
        None
    }

    pub fn move_down(&mut self) {
        if self.cursor + 1 < self.visible.len() { self.cursor += 1; }
    }

    pub fn move_up(&mut self) {
        if self.cursor > 0 { self.cursor -= 1; }
    }

    pub fn toggle_expand(&mut self) {
        let id = self.visible[self.cursor].clone();
        if self.collapsed.contains(&id) {
            self.collapsed.remove(&id);
        } else {
            self.collapsed.insert(id);
        }
        self.rebuild_visible();
    }

    pub fn is_expanded(&self) -> bool {
        !self.collapsed.contains(&self.visible[self.cursor])
    }

    pub fn navigate_to_parent(&mut self) {
        let selected_id = &self.visible[self.cursor];
        if let Some(parent) = self.find_parent_id(&self.root, selected_id) {
            if let Some(pos) = self.visible.iter().position(|id| id == &parent) {
                self.cursor = pos;
            }
        }
    }

    fn find_parent_id(&self, node: &EntityNode, child_id: &str) -> Option<String> {
        for child in &node.children {
            if child.id == child_id { return Some(node.id.clone()); }
            if let Some(found) = self.find_parent_id(child, child_id) { return Some(found); }
        }
        None
    }

    pub fn go_to_top(&mut self) { self.cursor = 0; }
    pub fn go_to_bottom(&mut self) {
        if !self.visible.is_empty() { self.cursor = self.visible.len() - 1; }
    }
}
```

- [ ] **Step 6: Write proper tree tests**

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::entity::*;

    fn make_test_tree() -> Tree {
        let n1 = Box::new(EntityNode::new("n1".into(), "JP-01".into(), "●".into(), EntityKind::Node, false, vec![], vec![]));
        let n2 = Box::new(EntityNode::new("n2".into(), "SG-02".into(), "●".into(), EntityKind::Node, false, vec![], vec![]));
        let sub = Box::new(EntityNode::new("sub".into(), "my-vpn".into(), ">".into(), EntityKind::Subscription, true, vec![n1, n2], vec![]));
        let sec = Box::new(EntityNode::new("sec".into(), "Subscriptions".into(), "~".into(), EntityKind::Section(SectionKind::Subscriptions), true, vec![sub], vec![]));
        Tree::new(sec)
    }

    #[test]
    fn test_initial_state() {
        let tree = make_test_tree();
        assert_eq!(tree.visible_len(), 1); // only the section node visible
        assert_eq!(tree.cursor(), 0);
        assert_eq!(tree.selected().id, "sec");
    }

    #[test]
    fn test_expand_reveals_children() {
        let mut tree = make_test_tree();
        tree.toggle_expand();
        assert_eq!(tree.visible_len(), 2); // sec + sub
        tree.move_down();
        tree.toggle_expand(); // expand sub
        assert_eq!(tree.visible_len(), 4); // sec + sub + n1 + n2
    }

    #[test]
    fn test_cursor_movement() {
        let mut tree = make_test_tree();
        tree.toggle_expand(); // sec
        tree.move_down(); // sub
        assert_eq!(tree.selected().id, "sub");
        tree.move_up(); // back to sec
        assert_eq!(tree.selected().id, "sec");
        tree.move_up(); // stays at sec (top boundary)
        assert_eq!(tree.selected().id, "sec");
        tree.move_down();
        tree.move_down(); // bottom boundary
        assert_eq!(tree.selected().id, "sub");
    }

    #[test]
    fn test_navigate_to_parent() {
        let mut tree = make_test_tree();
        tree.toggle_expand(); // sec
        tree.move_down(); // sub
        tree.toggle_expand(); // sub
        tree.move_down(); // n1
        tree.navigate_to_parent();
        assert_eq!(tree.selected().id, "sub");
    }

    #[test]
    fn test_collapse() {
        let mut tree = make_test_tree();
        tree.toggle_expand(); // expand sec
        assert_eq!(tree.visible_len(), 2);
        tree.toggle_expand(); // collapse sec
        assert_eq!(tree.visible_len(), 1);
    }
}
```

- [ ] **Step 7: Run all tests**

Run: `cd tui && cargo test`
Expected: all tests pass

- [ ] **Step 8: Commit**

```bash
cd tui && git add src/core/entity.rs src/core/tree.rs
git commit -m "feat(tui): add tree navigation with expand/collapse"
```

---

### Task 4: Base HTTP API client

**Files:**
- Create: `tui/src/core/api_client.rs`

- [ ] **Step 1: Write test for ApiClient URL construction**

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_build_url() {
        let client = ApiClient::new("http://localhost:8080".into());
        assert_eq!(client.url("/api/health"), "http://localhost:8080/api/health");
        assert_eq!(client.url("/api/subscription"), "http://localhost:8080/api/subscription");
    }

    #[test]
    fn test_build_url_with_trailing_slash() {
        let client = ApiClient::new("http://localhost:8080/".into());
        assert_eq!(client.url("/api/health"), "http://localhost:8080/api/health");
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tui && cargo test`
Expected: compile error — ApiClient not found

- [ ] **Step 3: Write api_client.rs implementation**

```rust
use anyhow::Result;
use reqwest::Client;

/// Base HTTP client for backend API.
pub struct ApiClient {
    base_url: String,
    client: Client,
}

impl ApiClient {
    pub fn new(base_url: String) -> Self {
        let base_url = base_url.trim_end_matches('/').to_string();
        Self { base_url, client: Client::new() }
    }

    pub fn url(&self, path: &str) -> String {
        let path = path.trim_start_matches('/');
        format!("{}/{}", self.base_url, path)
    }

    pub async fn get_json<T: serde::de::DeserializeOwned>(&self, path: &str) -> Result<T> {
        let resp = self.client.get(&self.url(path)).send().await?;
        if !resp.status().is_success() {
            anyhow::bail!("API error: {} {}", resp.status(), resp.text().await?);
        }
        Ok(resp.json::<T>().await?)
    }

    pub async fn post_json<T: serde::de::DeserializeOwned>(
        &self, path: &str, body: &impl serde::Serialize,
    ) -> Result<T> {
        let resp = self.client.post(&self.url(path)).json(body).send().await?;
        if !resp.status().is_success() {
            anyhow::bail!("API error: {} {}", resp.status(), resp.text().await?);
        }
        Ok(resp.json::<T>().await?)
    }

    pub async fn delete(&self, path: &str) -> Result<()> {
        let resp = self.client.delete(&self.url(path)).send().await?;
        if !resp.status().is_success() {
            anyhow::bail!("API error: {} {}", resp.status(), resp.text().await?);
        }
        Ok(())
    }

    pub async fn get_text(&self, path: &str) -> Result<String> {
        let resp = self.client.get(&self.url(path)).send().await?;
        if !resp.status().is_success() {
            anyhow::bail!("API error: {} {}", resp.status(), resp.text().await?);
        }
        Ok(resp.text().await?)
    }

    pub async fn post_raw(&self, path: &str, body: &str) -> Result<String> {
        let resp = self.client.post(&self.url(path))
            .header("Content-Type", "application/json")
            .body(body.to_owned())
            .send().await?;
        if !resp.status().is_success() {
            anyhow::bail!("API error: {} {}", resp.status(), resp.text().await?);
        }
        Ok(resp.text().await?)
    }
}
```

- [ ] **Step 4: Run tests**

Run: `cd tui && cargo test`
Expected: URL tests pass (API call tests would need a running server — skip for prototype)

- [ ] **Step 5: Commit**

```bash
cd tui && git add src/core/api_client.rs
git commit -m "feat(tui): add base HTTP API client"
```

---

### Task 5: Layout — split-panel rendering

**Files:**
- Create: `tui/src/core/layout.rs`

- [ ] **Step 1: Write layout.rs**

This file provides functions to render the left panel (tree), right panel (detail view), and status bar. It uses ratatui layout primitives.

```rust
use ratatui::{
    layout::{Constraint, Direction, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span, Text},
    widgets::{Block, Borders, Paragraph, Wrap},
    Frame,
};
use crate::core::tree::Tree;
use crate::core::entity::{EntityKind, SectionKind};

/// Renders the full screen layout.
pub fn render(frame: &mut Frame, tree: &Tree, right_content: &str, status_bar: &str) {
    let area = frame.area();

    // Split into main area + status bar
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(1), Constraint::Length(1)])
        .split(area);

    // Split main area into left panel + right panel
    let panels = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Percentage(30), Constraint::Percentage(70)])
        .split(chunks[0]);

    render_left_panel(frame, tree, panels[0]);
    render_right_panel(frame, right_content, panels[1]);
    render_status_bar(frame, status_bar, chunks[1]);
}

fn render_left_panel(frame: &mut Frame, tree: &Tree, area: Rect) {
    let visible = tree.visible_ids();
    let cursor = tree.cursor();
    let items: Vec<Line> = visible.iter().enumerate().map(|(i, id)| {
        let entity = tree.find_by_id(id).unwrap();
        let indent = if entity.is_section() { "" } else { "  " };
        let expanded = if entity.can_have_children && !tree.is_collapsed(id) { "▼ " } else if entity.can_have_children { "▶ " } else { "  " };
        let prefix = format!("{}{}{} ", indent, expanded, entity.icon);
        
        if i == cursor {
            Line::from(Span::styled(
                format!("{}{}", prefix, entity.label),
                Style::default().fg(Color::Black).bg(Color::White).add_modifier(Modifier::BOLD),
            ))
        } else {
            Line::from(Span::raw(format!("{}{}", prefix, entity.label)))
        }
    }).collect();

    let list = Paragraph::new(Text::from(items))
        .block(Block::default().borders(Borders::ALL).title("Navigation"));
    frame.render_widget(list, area);
}

fn render_right_panel(frame: &mut Frame, content: &str, area: Rect) {
    let paragraph = Paragraph::new(content)
        .block(Block::default().borders(Borders::ALL).title("Details"))
        .wrap(Wrap { trim: false });
    frame.render_widget(paragraph, area);
}

fn render_status_bar(frame: &mut Frame, text: &str, area: Rect) {
    let bar = Paragraph::new(Line::from(Span::styled(
        text,
        Style::default().fg(Color::White).bg(Color::DarkGray),
    )));
    frame.render_widget(bar, area);
}

// Helper on Tree to check collapsed state
impl Tree {
    pub fn is_collapsed(&self, id: &str) -> bool {
        self.visible_ids().contains(&id.to_string()) && !self.is_expanded_for_id(id)
    }
}
```

Keep layout.rs focused on rendering; add helpers to tree.rs.

- [ ] **Step 2: Add collapse helper to tree.rs**

```rust
/// Check if a specific entity is collapsed (even if not selected).
pub fn is_entity_collapsed(&self, id: &str) -> bool {
    self.collapsed.contains(id)
}
```

- [ ] **Step 3: Make find_by_id public for tree.rs**

In tree.rs, make `find_by_id` public:

```rust
pub fn find_by_id(&self, id: &str) -> Option<&EntityNode> {
    self.find_rec(&self.root, id)
}
```

- [ ] **Step 4: Commit**

```bash
cd tui && git add src/core/layout.rs src/core/tree.rs
git commit -m "feat(tui): add split-panel layout rendering"
```

---

### Task 6: Keybind dispatcher

**Files:**
- Create: `tui/src/core/keybind.rs`
- Create: `tui/src/core/input.rs`

- [ ] **Step 1: Write keybind.rs**

```rust
use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
use crate::core::tree::Tree;
use crate::core::entity::Action;

/// Modes the app can be in.
#[derive(Debug, Clone, PartialEq)]
pub enum AppMode {
    Normal,
    Input,
    Editor,
    Confirm,
}

/// Actions triggered by key events in Normal mode.
#[derive(Debug, Clone, PartialEq)]
pub enum NormalAction {
    NavUp,
    NavDown,
    GoToTop,
    GoToBottom,
    ExpandToggle,
    NavigateParent,
    Quit,
    ShowHelp,
    ExecuteCommand(Action),
}

/// Parse a key event in the given mode.
pub fn handle_key(
    key: KeyEvent,
    mode: &AppMode,
    tree: &Tree,
) -> Option<NormalAction> {
    if *mode != AppMode::Normal { return None; }

    match key.code {
        KeyCode::Up | KeyCode::Char('j') => Some(NormalAction::NavUp),
        KeyCode::Down | KeyCode::Char('k') => Some(NormalAction::NavDown),
        KeyCode::Right | KeyCode::Char('l') | KeyCode::Enter => Some(NormalAction::ExpandToggle),
        KeyCode::Left | KeyCode::Char('h') => Some(NormalAction::NavigateParent),
        KeyCode::Char('g') => {
            if key.modifiers == KeyModifiers::SHIFT {
                Some(NormalAction::GoToBottom)
            } else {
                Some(NormalAction::GoToTop)
            }
        }
        KeyCode::Char('q') => Some(NormalAction::Quit),
        KeyCode::Char('?') => Some(NormalAction::ShowHelp),
        KeyCode::Char(c) => {
            // Check if this key matches a command on the selected entity
            let entity = tree.selected();
            for cmd in &entity.commands {
                if cmd.key == c {
                    return Some(NormalAction::ExecuteCommand(cmd.action.clone()));
                }
            }
            None
        }
        _ => None,
    }
}

/// Build the status bar text showing available commands for the selected entity.
pub fn status_bar_commands(tree: &Tree) -> String {
    let entity = tree.selected();
    let cmds: Vec<String> = entity.commands.iter()
        .map(|c| format!("[{}] {}", c.key, c.label))
        .collect();
    if cmds.is_empty() {
        "[?] Help  [q] Quit".to_string()
    } else {
        format!("{}  [?] Help  [q] Quit", cmds.join("  "))
    }
}
```

- [ ] **Step 2: Write input.rs**

```rust
use ratatui::{
    style::{Color, Style},
    widgets::{Block, Borders, Paragraph},
    Frame,
};
use crate::core::keybind::AppMode;

/// State for a text input dialog.
pub struct InputState {
    pub prompt: String,
    pub value: String,
    pub cursor_pos: usize,
}

impl InputState {
    pub fn new(prompt: String) -> Self {
        Self { prompt, value: String::new(), cursor_pos: 0 }
    }
}

/// Render input overlay.
pub fn render_input(frame: &mut Frame, state: &InputState) {
    let area = frame.area();
    let input_area = ratatui::layout::Rect::new(
        area.width / 4,
        area.height / 2 - 1,
        area.width / 2,
        3,
    );

    let text = format!("{}: {}", state.prompt, state.value);
    let paragraph = Paragraph::new(text)
        .block(Block::default().borders(Borders::ALL).title("Input"))
        .style(Style::default().fg(Color::White));
    frame.render_widget(paragraph, input_area);
}
```

- [ ] **Step 3: Write tests for keybind**

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};

    #[test]
    fn test_nav_keys() {
        let tree = /* mock tree */;
        let key = KeyEvent::new(KeyCode::Up, KeyModifiers::NONE);
        assert_eq!(
            handle_key(key, &AppMode::Normal, &tree),
            Some(NormalAction::NavUp)
        );
    }
}
```

For the prototype, test command lookup logic rather than key parsing:

- [ ] **Step 4: Simplified keybind test**

```rust
#[cfg(test)]
mod tests {
    use super::::*;
    use crate::core::entity::*;

    #[test]
    fn test_status_bar_with_commands() {
        let n1 = Box::new(EntityNode::new("n1".into(), "N1".into(), "●".into(), EntityKind::Node, false, vec![], vec![
            Command::new('p', "Probe", Action::Probe),
        ]));
        let sub = Box::new(EntityNode::new("sub".into(), "Sub".into(), ">".into(), EntityKind::Subscription, true, vec![n1], vec![
            Command::new('r', "Refresh", Action::Refresh),
            Command::new('d', "Delete", Action::Delete),
        ]));
        let tree = crate::core::tree::Tree::new(sub);
        let bar = status_bar_commands(&tree);
        assert!(bar.contains("[r] Refresh"));
        assert!(bar.contains("[d] Delete"));
        assert!(bar.contains("[?] Help"));
    }
}
```

- [ ] **Step 5: Run tests**

Run: `cd tui && cargo test`
Expected: tests pass

- [ ] **Step 6: Commit**

```bash
cd tui && git add src/core/keybind.rs src/core/input.rs
git commit -m "feat(tui): add keybind dispatcher and input dialog"
```

---

### Task 7: App state + event loop

**Files:**
- Create: `tui/src/core/app.rs`

- [ ] **Step 1: Write app.rs**

```rust
use crate::core::tree::Tree;
use crate::core::keybind::{AppMode, NormalAction};
use crate::core::input::InputState;
use crate::core::entity::{EntityNode, Action, Command};
use crate::core::editor::Editor;
use std::collections::HashMap;

/// Confirm dialog state.
pub struct ConfirmState {
    pub message: String,
    pub on_confirm: Box<dyn FnOnce(&mut App)>,
}

/// Main application state.
pub struct App {
    pub tree: Tree,
    pub mode: AppMode,
    pub loading: bool,
    pub status_message: Option<String>,
    pub error_message: Option<String>,
    pub editor: Option<Editor>,
    pub input_state: Option<InputState>,
    pub confirm_state: Option<ConfirmState>,
    pub right_content: String,
}

impl App {
    pub fn new(root: Box<EntityNode>) -> Self {
        Self {
            tree: Tree::new(root),
            mode: AppMode::Normal,
            loading: false,
            status_message: None,
            error_message: None,
            editor_state: None,
            input_state: None,
            confirm_state: None,
            right_content: String::new(),
        }
    }

    pub fn set_status(&mut self, msg: String) {
        self.status_message = Some(msg);
        self.error_message = None;
    }

    pub fn set_error(&mut self, msg: String) {
        self.error_message = Some(msg);
        self.status_message = None;
    }

    /// Apply a normal-mode action to the app state.
    pub fn apply_action(&mut self, action: NormalAction) {
        match action {
            NormalAction::NavUp => self.tree.move_up(),
            NormalAction::NavDown => self.tree.move_down(),
            NormalAction::GoToTop => self.tree.go_to_top(),
            NormalAction::GoToBottom => self.tree.go_to_bottom(),
            NormalAction::ExpandToggle => self.tree.toggle_expand(),
            NormalAction::NavigateParent => self.tree.navigate_to_parent(),
            NormalAction::Quit => { /* handled in main loop */ }
            NormalAction::ShowHelp => {
                self.right_content = HELP_TEXT.to_string();
            }
            NormalAction::ExecuteCommand(action) => {
                // Dispatch to the appropriate handler — filled in by slices
                self.dispatch_action(action);
            }
        }
    }

    fn dispatch_action(&mut self, action: Action) {
        // Stub — slices will register their handlers here
        self.set_status(format!("Action: {:?} (not yet implemented)", action));
    }
}

const HELP_TEXT: &str = r#"singbox-ui TUI Help

Navigation:
  ↑/j    Move up
  ↓/k    Move down
  →/l    Expand/collapse section
  ←/h    Go to parent
  g      Go to top
  G      Go to bottom
  q      Quit
  ?      Toggle this help

Commands (context-sensitive):
  Press the key shown in the status bar to execute commands
  on the currently selected item.
"#;
```

- [ ] **Step 2: Commit**

```bash
cd tui && git add src/core/app.rs
git commit -m "feat(tui): add main App state and action dispatch"
```

---

### Task 8: Subscription slice — full implementation

**Files:**
- Create: `tui/src/slices/mod.rs` (Slice trait)
- Create: `tui/src/slices/subscription.rs`

- [ ] **Step 1: Write slices/mod.rs**

```rust
use crate::core::entity::EntityNode;
use crate::core::app::App;
use crate::core::api_client::ApiClient;
use crate::core::keybind::AppMode;
use std::sync::Arc;
use tokio::sync::Mutex;

/// A vertical slice: one domain module.
pub trait Slice {
    /// Name of this slice (matches section label).
    fn name(&self) -> &'static str;
    /// Build the root EntityNode for this slice (shown in navigation tree).
    fn build_entity(&self) -> Box<EntityNode>;
    /// Handle an action targeted at an entity in this slice.
    fn handle_action(&self, entity_id: &str, action: &crate::core::entity::Action, api: &ApiClient, app: &mut App);
}
```

- [ ] **Step 2: Write subscription.rs**

```rust
use crate::core::entity::*;
use crate::core::app::App;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;
use serde::Deserialize;

// ── API response types ──

#[derive(Debug, Deserialize)]
pub struct SubscriptionsResponse {
    pub subscriptions: Vec<SubscriptionEntry>,
    pub count: i64,
    pub total_nodes: i64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct SubscriptionEntry {
    pub id: String,
    pub name: String,
    pub url: String,
    pub user_agent: Option<String>,
    pub auto_update: Option<bool>,
    pub update_interval: Option<i64>,
    pub last_updated: Option<String>,
    pub nodes: Vec<ProxyNodeEntry>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ProxyNodeEntry {
    pub name: String,
    pub protocol: String,
    pub address: String,
    pub port: i64,
    pub latency: Option<i64>,
    pub online: Option<bool>,
    pub speed_kbps: Option<f64>,
}

// ── Slice implementation ──

pub struct SubscriptionSlice;

impl Slice for SubscriptionSlice {
    fn name(&self) -> &'static str { "Subscriptions" }

    fn build_entity(&self) -> Box<EntityNode> {
        Box::new(EntityNode::new(
            "section-sub".into(),
            "Subscriptions".into(),
            "~".into(),
            EntityKind::Section(SectionKind::Subscriptions),
            true,
            vec![], // children fetched dynamically
            vec![Command::new('a', "Add", Action::Add)],
        ))
    }

    fn handle_action(&self, entity_id: &str, action: &Action, api: &ApiClient, app: &mut App) {
        match action {
            Action::Refresh => {
                if entity_id == "section-sub" {
                    // Refresh all subscriptions
                    let api = api.clone();
                    app.loading = true;
                    tokio::spawn(async move {
                        match refresh_all_subscriptions(&api).await {
                            Ok(data) => {
                                // Update subscriptions in app state
                                // We need a shared state mechanism...
                            }
                            Err(e) => { eprintln!("Error: {}", e); }
                        }
                    });
                } else {
                    // Refresh single subscription
                }
            }
            _ => {}
        }
    }
}
```

**Approach for prototype:** 
- All requests are dispatched through `SliceRegistry::dispatch()` which returns `Result<String, String>`
- The main loop calls dispatch on key press, gets the view content string back
- Uses `reqwest::Client` (async) but `.await` blocks in the main loop (simplest for prototype)
- Data is shown as formatted text in the right panel

- [ ] **Step 3: Write subscription.rs with blocking API**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct SubscriptionSlice;

impl Slice for SubscriptionSlice {
    fn name(&self) -> &'static str { "Subscriptions" }

    fn build_entity(&self) -> Box<EntityNode> {
        Box::new(EntityNode::new(
            "section-sub".into(), "Subscriptions".into(), "~".into(),
            EntityKind::Section(SectionKind::Subscriptions), true, vec![], vec![
                Command::new('a', "Add", Action::Add),
                Command::new('r', "Refresh All", Action::Refresh),
            ],
        ))
    }

    fn handle_action(&self, entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match action {
            Action::Refresh if entity_id == "section-sub" => {
                let data: serde_json::Value = api.get_json("/api/subscription")
                    .map_err(|e| format!("API error: {}", e))?;
                let subs = &data["subscriptions"];
                let mut lines = vec![format!("Subscriptions ({} total)", data["count"])];
                lines.push("─".repeat(40));
                lines.push(format!("{:<20} {:>6} {:>10}", "Name", "Nodes", "Updated"));
                lines.push("─".repeat(40));
                if let Some(arr) = subs.as_array() {
                    for sub in arr {
                        let name = sub["name"].as_str().unwrap_or("?");
                        let nodes = sub["nodes"].as_array().map(|n| n.len()).unwrap_or(0);
                        let updated = sub["last_updated"].as_str().unwrap_or("never");
                        lines.push(format!("{:<20} {:>6} {:>10}", name, nodes, &updated[..10.min(updated.len())]));
                    }
                }
                Ok(lines.join("\n"))
            }
            _ => Err("Unknown action".into()),
        }
    }
}
```

Because the structure of App uses `right_content: String`, the slices return formatted strings. This keeps it simple for the prototype — no complex widget rendering.

- [ ] **Step 4: Write tests for subscription response parsing**

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_subscription_response() {
        let json = r#"{
            "subscriptions": [{
                "id": "sub_1",
                "name": "Test VPN",
                "url": "https://example.com/sub",
                "nodes": [{
                    "name": "JP-01",
                    "protocol": "shadowsocks",
                    "address": "1.2.3.4",
                    "port": 443,
                    "latency": 12,
                    "online": true,
                    "speed_kbps": 45000.0
                }]
            }],
            "count": 1,
            "totalNodes": 1
        }"#;
        let data: serde_json::Value = serde_json::from_str(json).unwrap();
        let subs = data["subscriptions"].as_array().unwrap();
        assert_eq!(subs.len(), 1);
        assert_eq!(subs[0]["name"], "Test VPN");
        assert_eq!(subs[0]["nodes"][0]["latency"], 12);
    }

    #[test]
    fn test_subscription_view_format() {
        // Test that the view formatting produces expected output
        let json = r#"{"subscriptions":[],"count":0,"totalNodes":0}"#;
        let data: serde_json::Value = serde_json::from_str(json).unwrap();
        let count = data["count"].as_i64().unwrap();
        assert_eq!(count, 0);
    }
}
```

- [ ] **Step 5: Commit**

```bash
cd tui && git add src/slices/mod.rs src/slices/subscription.rs
git commit -m "feat(tui): add Slice trait and Subscription slice"
```

---

### Task 9: Prober slice — full implementation

**Files:**
- Create: `tui/src/slices/prober.rs`

- [ ] **Step 1: Write prober.rs**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct ProberSlice;

impl Slice for ProberSlice {
    fn name(&self) -> &'static str { "Prober" }

    fn build_entity(&self) -> Box<EntityNode> {
        let status_entity = Box::new(EntityNode::new(
            "prober-status".into(), "Status".into(), "●".into(),
            EntityKind::Status, false, vec![], vec![
                Command::new('s', "Start/Stop", Action::Start),
                Command::new('r', "Refresh", Action::Refresh),
            ],
        ));
        Box::new(EntityNode::new(
            "section-prober".into(), "Prober".into(), "~".into(),
            EntityKind::Section(SectionKind::Prober), true, vec![status_entity], vec![
                Command::new('y', "Sync from subs", Action::Sync),
            ],
        ))
    }

    fn handle_action(&self, entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match (entity_id, action) {
            ("prober-status", Action::Refresh) | ("section-prober", Action::Refresh) => {
                let status: serde_json::Value = api.get_json("/api/prober/status")
                    .map_err(|e| format!("API error: {}", e))?;
                let results: serde_json::Value = api.get_json("/api/prober/results")
                    .map_err(|e| format!("API error: {}", e))?;
                
                let mut lines = vec![];
                lines.push(format!("Prober Status: {}", status.get("running").and_then(|v| v.as_bool()).map(|b| if b { "RUNNING" } else { "STOPPED" }).unwrap_or("?")));
                lines.push(format!("Total nodes: {}", status.get("total_nodes").and_then(|v| v.as_i64()).unwrap_or(0)));
                lines.push("".into());
                lines.push("─ Results ─────────────────".into());
                
                if let Some(results_arr) = results["results"].as_array() {
                    for r in results_arr {
                        let tag = r["nodeTag"].as_str().unwrap_or("?");
                        let lat = r["latency"].as_i64().unwrap_or(0);
                        let status_str = r["status"].as_str().unwrap_or("?");
                        let sr = r["successRate"].as_f64().unwrap_or(0.0);
                        lines.push(format!("  {:<20} {:>6}ms  {}  {:.0}%", tag, lat, status_str, sr * 100.0));
                    }
                }
                Ok(lines.join("\n"))
            }
            ("prober-status", Action::Start) => {
                api.post_json::<serde_json::Value>("/api/prober/start", &{})
                    .map_err(|e| format!("API error: {}", e))?;
                Ok("Prober started".into())
            }
            ("prober-status", Action::Stop) => {
                api.post_json::<serde_json::Value>("/api/prober/stop", &{})
                    .map_err(|e| format!("API error: {}", e))?;
                Ok("Prober stopped".into())
            }
            _ => Err("Unknown action".into()),
        }
    }
}
```

Use `Action::Start` to toggle start/stop based on current status:

```rust
("prober-status", Action::Start) => {
    // Get current status first
    let status: serde_json::Value = api.get_json("/api/prober/status")
        .map_err(|e| format!("API error: {}", e))?;
    let running = status.get("running").and_then(|v| v.as_bool()).unwrap_or(false);
    if running {
        api.post_json::<serde_json::Value>("/api/prober/stop", &{}).map_err(|e| format!("API error: {}", e))?;
        Ok("Prober stopped".into())
    } else {
        api.post_json::<serde_json::Value>("/api/prober/start", &{}).map_err(|e| format!("API error: {}", e))?;
        Ok("Prober started".into())
    }
}
```

- [ ] **Step 2: Write tests**

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_prober_results() {
        let json = r#"{"results":[{"nodeTag":"JP-01","latency":12,"status":"online","successRate":1.0}]}"#;
        let data: serde_json::Value = serde_json::from_str(json).unwrap();
        let results = data["results"].as_array().unwrap();
        assert_eq!(results[0]["nodeTag"], "JP-01");
        assert_eq!(results[0]["latency"], 12);
    }
}
```

- [ ] **Step 3: Commit**

```bash
cd tui && git add src/slices/prober.rs
git commit -m "feat(tui): add Prober slice"
```

---

### Task 10: Singbox slice — full implementation

**Files:**
- Create: `tui/src/slices/singbox.rs`

- [ ] **Step 1: Write singbox.rs**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct SingboxSlice;

impl Slice for SingboxSlice {
    fn name(&self) -> &'static str { "Singbox" }

    fn build_entity(&self) -> Box<EntityNode> {
        let config_entity = Box::new(EntityNode::new(
            "sb-config".into(), "Config".into(), "●".into(),
            EntityKind::Config, false, vec![], vec![
                Command::new('e', "Edit JSON", Action::EditConfig),
                Command::new('r', "Refresh", Action::Refresh),
            ],
        ));
        let logs_entity = Box::new(EntityNode::new(
            "sb-logs".into(), "Logs".into(), "●".into(),
            EntityKind::Logs, false, vec![], vec![
                Command::new('l', "View Logs", Action::ViewLogs),
            ],
        ));
        Box::new(EntityNode::new(
            "section-singbox".into(), "Singbox".into(), "~".into(),
            EntityKind::Section(SectionKind::Singbox), true, vec![config_entity, logs_entity], vec![
                Command::new('r', "Refresh", Action::Refresh),
            ],
        ))
    }

    fn handle_action(&self, entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match (entity_id, action) {
            ("section-singbox", Action::Refresh) => {
                let version: serde_json::Value = api.get_json("/api/singbox/version")
                    .map_err(|e| format!("API error: {}", e))?;
                let status: serde_json::Value = api.get_json("/api/singbox/status")
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(format!(
                    "Sing-box v{}\n\nStatus: {}\nContainer: {}",
                    version["version"].as_str().unwrap_or("?"),
                    if status["running"].as_bool().unwrap_or(false) { "RUNNING" } else { "STOPPED" },
                    status["containerId"].as_str().unwrap_or("-"),
                ))
            }
            ("sb-config", Action::Refresh) => {
                let config = api.get_text("/api/singbox/config")
                    .map_err(|e| format!("API error: {}", e))?;
                // Pretty-print
                if let Ok(v) = serde_json::from_str::<serde_json::Value>(&config) {
                    serde_json::to_string_pretty(&v).map_err(|e| format!("JSON error: {}", e))
                } else {
                    Ok(config)
                }
            }
            ("sb-config", Action::EditConfig) => {
                // Return the current config text — the app will open the editor overlay
                let config = api.get_text("/api/singbox/config")
                    .map_err(|e| format!("API error: {}", e))?;
                if let Ok(v) = serde_json::from_str::<serde_json::Value>(&config) {
                    serde_json::to_string_pretty(&v).map_err(|e| format!("JSON error: {}", e))
                } else {
                    Ok(config)
                }
            }
            ("sb-logs", Action::ViewLogs) | ("sb-logs", Action::Refresh) => {
                let logs: serde_json::Value = api.get_json("/api/singbox/logs")
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(logs["logs"].as_str().unwrap_or("No logs").to_string())
            }
            _ => Err("Unknown action".into()),
        }
    }
}
```

- [ ] **Step 2: Write tests**

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_version() {
        let json = r#"{"version":"1.10.0"}"#;
        let data: serde_json::Value = serde_json::from_str(json).unwrap();
        assert_eq!(data["version"], "1.10.0");
    }

    #[test]
    fn test_parse_status() {
        let json = r#"{"running":true,"containerId":"abc123"}"#;
        let data: serde_json::Value = serde_json::from_str(json).unwrap();
        assert!(data["running"].as_bool().unwrap());
        assert_eq!(data["containerId"], "abc123");
    }
}
```

- [ ] **Step 3: Commit**

```bash
cd tui && git add src/slices/singbox.rs
git commit -m "feat(tui): add Singbox slice with config/logs"
```

---

### Task 11: Stub slices — speedtest, wireguard, warp, certificate

**Files:**
- Create: `tui/src/slices/speedtest.rs`
- Create: `tui/src/slices/wireguard.rs`
- Create: `tui/src/slices/warp.rs`
- Create: `tui/src/slices/certificate.rs`

- [ ] **Step 1: Write speedtest.rs**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct SpeedtestSlice;

impl Slice for SpeedtestSlice {
    fn name(&self) -> &'static str { "Speedtest" }

    fn build_entity(&self) -> Box<EntityNode> {
        Box::new(EntityNode::new(
            "section-speedtest".into(), "Speedtest".into(), "~".into(),
            EntityKind::Section(SectionKind::Speedtest), false, vec![], vec![
                Command::new('s', "Start", Action::Start),
            ],
        ))
    }

    fn handle_action(&self, _entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match action {
            Action::Start => {
                api.post_json::<serde_json::Value>("/api/speedtest/start", &{})
                    .map_err(|e| format!("API error: {}", e))?;
                Ok("Speed test started".into())
            }
            Action::Refresh => {
                let status: serde_json::Value = api.get_json("/api/speedtest/status")
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(format!("Speedtest Status:\n\n{:#}", status))
            }
            _ => Err("Not implemented".into()),
        }
    }
}
```

- [ ] **Step 2: Write wireguard.rs**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct WireGuardSlice;

impl Slice for WireGuardSlice {
    fn name(&self) -> &'static str { "WireGuard" }

    fn build_entity(&self) -> Box<EntityNode> {
        Box::new(EntityNode::new(
            "section-wg".into(), "WireGuard".into(), "~".into(),
            EntityKind::Section(SectionKind::WireGuard), false, vec![], vec![
                Command::new('g', "Generate Keys", Action::GenerateKeys),
                Command::new('c', "Client Config", Action::Custom("client-config".into())),
            ],
        ))
    }

    fn handle_action(&self, _entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match action {
            Action::GenerateKeys => {
                let keys: serde_json::Value = api.post_json("/api/wireguard/keygen", &serde_json::json!({}))
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(format!(
                    "WireGuard Keys\n\nPrivate Key: {}\nPublic Key: {}\nIP: {}",
                    keys["private_key"].as_str().unwrap_or("?"),
                    keys["public_key"].as_str().unwrap_or("?"),
                    keys["ip"].as_str().unwrap_or("-"),
                ))
            }
            Action::Custom(cmd) if cmd == "client-config" => {
                let config = api.get_text("/api/wireguard/client-config")
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(config)
            }
            _ => Err("Not implemented".into()),
        }
    }
}
```

- [ ] **Step 3: Write warp.rs**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct WARPSlice;

impl Slice for WARPSlice {
    fn name(&self) -> &'static str { "WARP" }

    fn build_entity(&self) -> Box<EntityNode> {
        Box::new(EntityNode::new(
            "section-warp".into(), "WARP".into(), "~".into(),
            EntityKind::Section(SectionKind::WARP), false, vec![], vec![
                Command::new('n', "Register", Action::Register),
                Command::new('l', "Bind License", Action::BindLicense),
                Command::new('s', "Scan", Action::Scan),
            ],
        ))
    }

    fn handle_action(&self, _entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match action {
            Action::Register => {
                let resp: serde_json::Value = api.post_json("/api/warp/register", &{})
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(format!("WARP Registered\n\n{:#}", resp))
            }
            Action::Refresh => {
                let account = api.get_json::<serde_json::Value>("/api/warp/account")
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(format!("WARP Account\n\n{:#}", account))
            }
            _ => Err("Not implemented".into()),
        }
    }
}
```

- [ ] **Step 4: Write certificate.rs**

```rust
use crate::core::entity::*;
use crate::core::api_client::ApiClient;
use crate::slices::Slice;

pub struct CertificateSlice;

impl Slice for CertificateSlice {
    fn name(&self) -> &'static str { "Certificate" }

    fn build_entity(&self) -> Box<EntityNode> {
        Box::new(EntityNode::new(
            "section-cert".into(), "Certificate".into(), "~".into(),
            EntityKind::Section(SectionKind::Certificate), false, vec![], vec![
                Command::new('r', "Refresh", Action::Refresh),
            ],
        ))
    }

    fn handle_action(&self, _entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String> {
        match action {
            Action::Refresh => {
                let info = api.get_json::<serde_json::Value>("/api/singbox/certificate")
                    .map_err(|e| format!("API error: {}", e))?;
                Ok(format!("Certificate Info\n\n{:#}", info))
            }
            _ => Err("Not implemented".into()),
        }
    }
}
```

- [ ] **Step 5: Commit**

```bash
cd tui && git add src/slices/speedtest.rs src/slices/wireguard.rs src/slices/warp.rs src/slices/certificate.rs
git commit -m "feat(tui): add stub slices for speedtest, wireguard, warp, certificate"
```

---

### Task 12: Assembly — connecting core + slices

**Files:**
- Modify: `tui/src/slices/mod.rs` (add SliceRegistry)
- Modify: `tui/src/core/app.rs` (add slice integration, editor, confirm)
- Create: `tui/src/app.rs` (top-level assembly)

- [ ] **Step 1: Update slices/mod.rs with registry**

```rust
pub mod subscription;
pub mod prober;
pub mod singbox;
pub mod speedtest;
pub mod wireguard;
pub mod warp;
pub mod certificate;

use crate::core::entity::*;
use crate::core::api_client::ApiClient;

/// A vertical slice: one domain module.
pub trait Slice {
    fn name(&self) -> &'static str;
    fn build_entity(&self) -> Box<EntityNode>;
    fn handle_action(&self, entity_id: &str, action: &Action, api: &ApiClient) -> Result<String, String>;
}

/// Registry of all slices.
pub struct SliceRegistry {
    pub slices: Vec<Box<dyn Slice>>,
    pub api: ApiClient,
}

impl SliceRegistry {
    pub fn new(api: ApiClient) -> Self {
        Self { slices: vec![], api }
    }

    pub fn register(&mut self, slice: Box<dyn Slice>) {
        self.slices.push(slice);
    }

    /// Build the root entity tree from all registered slices.
    pub fn build_root(&self) -> Box<EntityNode> {
        let mut children = vec![];
        for slice in &self.slices {
            children.push(slice.build_entity());
        }
        Box::new(EntityNode::new(
            "root".into(), "singbox-ui".into(), "".into(),
            EntityKind::Root, true, children, vec![],
        ))
    }

    /// Find which slice handles an entity ID and dispatch.
    pub fn dispatch(&self, entity_id: &str, action: &Action) -> Result<String, String> {
        // Slice entity IDs are prefixed with "section-", but children share the slice
        let prefix = if entity_id.starts_with("section-") || entity_id.starts_with("sb-") || entity_id.starts_with("prober-") {
            entity_id.split('-').next().unwrap_or("")
        } else {
            // Try to find the owning slice by walking up the tree
            ""
        };
        
        // Simple approach: try each slice's entity tree to find a match
        for slice in &self.slices {
            let root = slice.build_entity();
            if contains_id(&root, entity_id) {
                return slice.handle_action(entity_id, action, &self.api);
            }
        }
        Err("No slice handles this entity".into())
    }
}

fn contains_id(node: &EntityNode, id: &str) -> bool {
    if node.id == id { return true; }
    node.children.iter().any(|child| contains_id(child, id))
}
```

- [ ] **Step 2: Update core/app.rs with slice integration**

Add to App:

```rust
pub fn update_right_content(&mut self, content: String) {
    self.right_content = content;
}
```

- [ ] **Step 3: Write app.rs (top-level assembly)**

```rust
mod core;
mod slices;

use core::app::App;
use core::api_client::ApiClient;
use core::keybind::{AppMode, NormalAction, status_bar_commands};
use core::layout;
use core::input;
use core::editor;
use slices::{SliceRegistry, Slice};
use slices::subscription::SubscriptionSlice;
use slices::prober::ProberSlice;
use slices::singbox::SingboxSlice;
use slices::speedtest::SpeedtestSlice;
use slices::wireguard::WireGuardSlice;
use slices::warp::WARPSlice;
use slices::certificate::CertificateSlice;

use ratatui::{
    backend::CrosstermBackend,
    Terminal,
};
use crossterm::{
    event::{self, DisableMouseCapture, EnableMouseCapture, Event, KeyEventKind},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use std::io;
use std::sync::Arc;
use tokio::sync::Mutex;

async fn run_app(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    mut app: App,
    registry: Arc<Mutex<SliceRegistry>>,
) -> anyhow::Result<()> {
    loop {
        // Render
        terminal.draw(|frame| {
            let status = status_bar_for_app(&app);
            
            if app.mode == AppMode::Input {
                if let Some(ref input_state) = app.input_state {
                    // Render with input overlay
                    layout::render(frame, &app.tree, &app.right_content, &status);
                    input::render_input(frame, input_state);
                    return;
                }
            }
            
            layout::render(frame, &app.tree, &app.right_content, &status);
        })?;

        // Handle events
        if let Event::Key(key) = event::read()? {
            if key.kind != KeyEventKind::Press { continue; }

            match app.mode {
                AppMode::Normal => {
                    if let Some(action) = core::keybind::handle_key(key, &app.mode, &app.tree) {
                        match action {
                            NormalAction::Quit => break,
                            NormalAction::ExecuteCommand(action) => {
                                let entity_id = app.tree.selected().id.clone();
                                let registry = registry.clone();
                                let result = registry.lock().await.dispatch(&entity_id, &action);
                                match result {
                                    Ok(content) => { app.update_right_content(content); }
                                    Err(e) => { app.set_error(e); }
                                }
                            }
                            other => { app.apply_action(other); }
                        }
                    }
                }
                AppMode::Input => {
                    // Handle input mode keys
                    match key.code {
                        crossterm::event::KeyCode::Esc => { app.mode = AppMode::Normal; }
                        crossterm::event::KeyCode::Enter => {
                            // Submit input
                            if let Some(input) = app.input_state.take() {
                                app.set_status(format!("Input: {}", input.value));
                            }
                            app.mode = AppMode::Normal;
                        }
                        crossterm::event::KeyCode::Backspace => {
                            if let Some(ref mut input) = app.input_state {
                                input.value.pop();
                            }
                        }
                        crossterm::event::KeyCode::Char(c) => {
                            if let Some(ref mut input) = app.input_state {
                                input.value.push(c);
                            }
                        }
                        _ => {}
                    }
                }
                _ => {}
            }
        }
    }
    Ok(())
}

fn status_bar_for_app(app: &App) -> String {
    let cmd_bar = status_bar_commands(&app.tree);
    if let Some(ref msg) = app.status_message {
        format!("{}  |  {}", cmd_bar, msg)
    } else if let Some(ref err) = app.error_message {
        format!("{}  |  ERROR: {}", cmd_bar, err)
    } else {
        cmd_bar
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Setup terminal
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    // Build app
    let api = ApiClient::new("http://localhost:8080".into());
    let mut registry = SliceRegistry::new(api);
    registry.register(Box::new(SubscriptionSlice));
    registry.register(Box::new(ProberSlice));
    registry.register(Box::new(SingboxSlice));
    registry.register(Box::new(SpeedtestSlice));
    registry.register(Box::new(WireGuardSlice));
    registry.register(Box::new(WARPSlice));
    registry.register(Box::new(CertificateSlice));

    let root = registry.build_root();
    let app = App::new(root);
    let registry = Arc::new(Mutex::new(registry));

    // Run
    let result = run_app(&mut terminal, app, registry).await;

    // Cleanup
    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen, DisableMouseCapture)?;
    terminal.show_cursor()?;

    result
}
```

All assembly logic goes into `main.rs`:
- `tui/src/core/app.rs` → App struct, modes, state  
- `tui/src/main.rs` → run_app(), main() entry point, terminal setup, event loop

- [ ] **Step 4: Write main.rs as the assembly**

```rust
mod core;
mod slices;

use core::app::App;
use core::api_client::ApiClient;
use core::keybind::{AppMode, NormalAction, status_bar_commands};
use core::layout;
use core::input;
use slices::{SliceRegistry, Slice};
use slices::subscription::SubscriptionSlice;
use slices::prober::ProberSlice;
use slices::singbox::SingboxSlice;
use slices::speedtest::SpeedtestSlice;
use slices::wireguard::WireGuardSlice;
use slices::warp::WARPSlice;
use slices::certificate::CertificateSlice;

use ratatui::{backend::CrosstermBackend, Terminal};
use crossterm::{
    event::{self, DisableMouseCapture, EnableMouseCapture, Event, KeyEventKind},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use std::io;
use std::sync::Arc;
use tokio::sync::Mutex;

async fn run_app(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    mut app: App,
    registry: Arc<Mutex<SliceRegistry>>,
) -> anyhow::Result<()> {
    loop {
        // Render
        terminal.draw(|frame| {
            let status = status_bar_for_app(&app);
            layout::render(frame, &app.tree, &app.right_content, &status);
        })?;

        // Handle events
        if let Event::Key(key) = event::read()? {
            if key.kind != KeyEventKind::Press { continue; }

            match app.mode {
                AppMode::Normal => {
                    if let Some(action) = core::keybind::handle_key(key, &app.mode, &app.tree) {
                        match action {
                            NormalAction::Quit => break,
                            NormalAction::ExecuteCommand(action) => {
                                let entity_id = app.tree.selected().id.clone();
                                let registry_clone = registry.clone();
                                let result = registry_clone.lock().await.dispatch(&entity_id, &action);
                                match result {
                                    Ok(content) => { app.right_content = content; }
                                    Err(e) => { app.set_error(e); }
                                }
                            }
                            other => { app.apply_action(other); }
                        }
                    }
                }
                AppMode::Input => {
                    match key.code {
                        crossterm::event::KeyCode::Esc => { app.mode = AppMode::Normal; }
                        crossterm::event::KeyCode::Enter => {
                            app.input_state.take();
                            app.mode = AppMode::Normal;
                        }
                        crossterm::event::KeyCode::Backspace => {
                            if let Some(ref mut input) = app.input_state {
                                input.value.pop();
                            }
                        }
                        crossterm::event::KeyCode::Char(c) => {
                            if let Some(ref mut input) = app.input_state {
                                input.value.push(c);
                            }
                        }
                        _ => {}
                    }
                }
                _ => {}
            }
        }
    }
    Ok(())
}

fn status_bar_for_app(app: &App) -> String {
    let cmd_bar = status_bar_commands(&app.tree);
    if let Some(ref msg) = app.status_message {
        format!("{} | {}", cmd_bar, msg)
    } else if let Some(ref err) = app.error_message {
        format!("{} | ERROR: {}", cmd_bar, err)
    } else {
        cmd_bar
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let api = ApiClient::new("http://localhost:8080".into());
    let mut registry = SliceRegistry::new(api);
    registry.register(Box::new(SubscriptionSlice));
    registry.register(Box::new(ProberSlice));
    registry.register(Box::new(SingboxSlice));
    registry.register(Box::new(SpeedtestSlice));
    registry.register(Box::new(WireGuardSlice));
    registry.register(Box::new(WARPSlice));
    registry.register(Box::new(CertificateSlice));

    let root = registry.build_root();
    let app = App::new(root);
    let registry = Arc::new(Mutex::new(registry));

    let result = run_app(&mut terminal, app, registry).await;

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen, DisableMouseCapture)?;
    terminal.show_cursor()?;

    result
}
```

- [ ] **Step 5: Try to compile and fix any errors**

Run: `cd tui && cargo build 2>&1`
Expected: compiles with minimal warnings

- [ ] **Step 6: Fix compilation issues**

Iterate on any compilation errors. Common issues:
- `serde_json::json!({})` needs serde_json import
- Module paths need `pub mod` or `mod` declarations
- Missing `use` statements
- `Clone` not derived for some types used with Arc<Mutex<>>

- [ ] **Step 7: Commit**

```bash
cd tui && git add src/ src/main.rs
git commit -m "feat(tui): assemble core + slices into working TUI prototype"
```

---

### Task 13: Editor overlay (tui-textarea)

**Files:**
- Create: `tui/src/core/editor.rs`

- [ ] **Step 1: Write editor.rs**

```rust
use ratatui::{
    layout::Rect,
    style::{Color, Style},
    widgets::Block,
    Frame,
};
use tui_textarea::TextArea;

/// State for the fullscreen JSON editor.
pub struct Editor {
    pub textarea: TextArea<'static>,
    pub title: String,
}

impl Editor {
    pub fn new(content: String, title: String) -> Self {
        let mut textarea = TextArea::default();
        textarea.insert_str(&content);
        textarea.set_block(
            Block::bordered().title(format!(" {} ", title)).title_bottom(" Ctrl+S: Save  Esc: Cancel "),
        );
        textarea.set_style(Style::default().fg(Color::White));
        Self { textarea, title }
    }

    /// Render the editor (fullscreen overlay).
    pub fn render(&mut self, frame: &mut Frame) {
        frame.render_widget(&self.textarea, frame.area());
    }

    pub fn handle_key(&mut self, key: crossterm::event::KeyEvent) -> EditorAction {
        match key.code {
            crossterm::event::KeyCode::Esc => EditorAction::Cancel,
            crossterm::event::KeyCode::Char('s') if key.modifiers.contains(crossterm::event::KeyModifiers::CONTROL) => {
                EditorAction::Save(self.textarea.lines().join("\n"))
            }
            _ => {
                self.textarea.input(key);
                EditorAction::Continue
            }
        }
    }
}

pub enum EditorAction {
    Continue,
    Save(String),
    Cancel,
}
```

- [ ] **Step 2: Integrate editor into App state**

Add `editor_state: Option<Editor>` to App struct (instead of the previous EditorState). Add editor handling to main.rs event loop.

- [ ] **Step 3: Commit**

```bash
cd tui && git add src/core/editor.rs
git commit -m "feat(tui): add JSON editor overlay with tui-textarea"
```

---

### Task 14: Final polish and test

- [ ] **Step 1: Run full test suite**

Run: `cd tui && cargo test`
Expected: all tests pass

- [ ] **Step 2: Fix any clippy warnings**

Run: `cd tui && cargo clippy 2>&1`
Fix any warnings.

- [ ] **Step 3: Final build**

Run: `cd tui && cargo build`
Expected: clean build

- [ ] **Step 4: Commit all remaining changes**

```bash
cd tui && git add -A && git commit -m "chore(tui): final polish and cleanup"
```
