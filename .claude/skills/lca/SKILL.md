---
name: lca
description: >
  Layered Composition Architecture patterns. Use when designing system
  boundaries, state flow, lifecycle management, or configuration.
  Triggers: System interface, New* constructor, Start() method, Cold Start,
  Hot Start, Runtime struct, Domain struct, lifecycle coordinator, OnStartup,
  OnShutdown, config.go, Finalize(), validate(), loadDefaults(), loadEnv().
  File patterns: internal/config/*.go, internal/lifecycle/*.go, cmd/server/*.go
---

# Layered Composition Architecture (LCA)

## When This Skill Applies

- Designing new systems or domains
- Implementing lifecycle management
- Creating configuration structures
- Understanding dependency flow
- Building runtime/domain separation

## Principles

### 1. State Flows Down, Never Up

State flows through method calls (parameters), not through object initialization.

**Anti-Pattern** (Reaching Up):
```go
type Handler struct {
    service *Service  // Storing reference to parent
}

func (h *Handler) Process() {
    sys := h.service.Providers()  // Reaching up to parent state
}
```

**Correct Pattern** (State Flows Down):
```go
func HandleCreate(w http.ResponseWriter, r *http.Request, system providers.System, logger *slog.Logger) {
    // State injected at call site, flows DOWN
}
```

### 2. Systems, Not Services/Models

**Terminology**:
- **System**: A cohesive unit that owns both state and processes
- **State**: Structures that define data
- **Process**: Methods that operate on state
- **Interface**: Contract between systems

**Package Organization**:
- **cmd/server**: The process (composition root, entry point)
- **pkg/**: Public API (shared infrastructure, reusable toolkit)
- **internal/**: Private API (domain systems, business logic)

### 3. Cold Start vs Hot Start

**Cold Start** (State Initialization):
- `New*()` constructor functions
- Builds entire dependency graph
- All configurations → State objects
- All systems created but dormant
- No processes running
- Returns ready-to-start system

**Hot Start** (Process Activation):
- `Start()` methods
- State objects → Running processes
- Cascade start through dependency graph
- Context boundaries for lifecycle management
- System becomes interactable

```go
svc, err := NewService(cfg)  // Cold Start - Build state graph
if err := svc.Start(); err != nil {  // Hot Start - Activate processes
    log.Fatal(err)
}
```

### 4. System Interface Contract

Every system provides:

1. **Internal State** (private) - Only accessible within the system
2. **Internal Processes** (private) - Implementation details
3. **Getter Methods** (public) - Immutable access to state
4. **Commands** (public) - Write operations from owner
5. **Events** (public, optional) - Notifications to owner

### 5. Runtime/Domain Separation

| Category | Characteristics | Examples |
|----------|----------------|----------|
| **Runtime Systems** | Long-running, lifecycle-managed, application-scoped | Database, Storage |
| **Domain Systems** | Stateless, request-scoped behavior, no lifecycle | Providers, Agents |

**Runtime Structure**:
```go
type Runtime struct {
    Lifecycle  *lifecycle.Coordinator
    Logger     *slog.Logger
    Database   database.System
    Storage    storage.System
    Pagination pagination.Config
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
    lc := lifecycle.New()
    logger := newLogger(&cfg.Logging)
    // Initialize systems...
    return &Runtime{...}, nil
}

func (r *Runtime) Start() error {
    if err := r.Database.Start(r.Lifecycle); err != nil {
        return fmt.Errorf("database start failed: %w", err)
    }
    return nil
}
```

### 6. Lifecycle Coordinator

```go
type Coordinator struct {
    ctx        context.Context
    cancel     context.CancelFunc
    startupWg  sync.WaitGroup
    shutdownWg sync.WaitGroup
    ready      bool
}

func (c *Coordinator) OnStartup(fn func())   // Register startup tasks
func (c *Coordinator) OnShutdown(fn func())  // Register cleanup tasks
func (c *Coordinator) WaitForStartup()       // Block until ready
func (c *Coordinator) Ready() bool           // Check readiness
```

**Usage Pattern**:
- **OnStartup**: Tasks that must complete for service readiness (e.g., database ping)
- **OnShutdown**: Cleanup tasks triggered on context cancellation (e.g., close connections)

### 7. Configuration Pattern

**Precedence** (highest to lowest):
```
Environment Variables
    ↓ replaces (not merges)
config.{env}.toml (overlay)
    ↓ replaces (not merges)
config.toml (base)
```

**Finalize Pattern**:
```go
type Config struct {
    Server   ServerConfig   `toml:"server"`
    Database DatabaseConfig `toml:"database"`
}

func (c *Config) Finalize() error {
    c.loadDefaults()  // Apply defaults for zero-value fields
    c.loadEnv()       // Map SECTION_FIELD environment variables
    return c.validate()  // Validate field constraints
}
```

**Section Config**:
```go
type ServerConfig struct {
    Host string `toml:"host"`
    Port int    `toml:"port"`
}

func (c *ServerConfig) loadDefaults() {
    if c.Host == "" { c.Host = "0.0.0.0" }
    if c.Port == 0 { c.Port = 8080 }
}

func (c *ServerConfig) loadEnv() {
    if v := os.Getenv("SERVER_HOST"); v != "" { c.Host = v }
    if v := os.Getenv("SERVER_PORT"); v != "" {
        if port, err := strconv.Atoi(v); err == nil { c.Port = port }
    }
}

func (c *ServerConfig) validate() error {
    if c.Port < 1 || c.Port > 65535 {
        return errors.New("port must be between 1 and 65535")
    }
    return nil
}
```

## Anti-Patterns

### Reaching Up for State

```go
// Bad: Handler reaches up to get dependencies
type Handler struct {
    app *Application
}

func (h *Handler) Process() {
    db := h.app.Database()  // Reaching up
}

// Good: Dependencies injected at call site
func HandleProcess(db database.System) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // db available directly
    }
}
```

### Merged Configuration

```go
// Bad: Merging arrays from multiple sources
defaults := []string{"a", "b"}
overrides := []string{"c"}
result := append(defaults, overrides...)  // ["a", "b", "c"]

// Good: Complete replacement at each level
result := overrides  // ["c"] - override replaces entirely
```
