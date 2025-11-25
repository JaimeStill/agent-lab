# agent-lab: A Fresh Take on Go Web Service Architecture

*Pre-release Technical Preview*

---

You know that feeling when you inherit a codebase and spend the first week just figuring out where things live? Where does the database connection come from? Why is this handler reaching into three different packages to get what it needs? What happens when the service shuts down?

**agent-lab** started as a platform for building agentic AI workflows, but along the way it became an exercise in answering a deeper question: *What does a clean, maintainable Go web service actually look like?*

Here's what we came up with.

---

## The Big Idea: Systems, Not Services

Most Go projects end up with a `/services` directory full of structs that do... things. And a `/models` directory with data structures. And handlers that reach up into services to get what they need.

We took a different approach. Everything is a **System** - a cohesive unit that owns both its state and the processes that operate on that state. A System exposes a clear interface: getters (nouns), commands (verbs), and optionally events (notifications).

```go
type System interface {
    // Getters (nouns) - access state
    Connection() *sql.DB
    Logger() *slog.Logger

    // Commands (verbs) - do things
    Start(lc *lifecycle.Coordinator) error
    Create(ctx context.Context, cmd CreateCommand) (*Provider, error)

    // Events (On*) - notifications
    OnShutdown() <-chan struct{}
    OnError() <-chan error
}
```

This isn't just naming convention. It changes how you think about the code.

---

## Runtime vs Domain: A Clear Separation

Here's where it gets interesting. We discovered that systems fall into two distinct categories:

| Category | Characteristics | Examples |
|----------|----------------|----------|
| **Runtime Systems** | Long-running, lifecycle-managed, application-scoped | Database, Server, Logger |
| **Domain Systems** | Stateless, request-scoped behavior, no lifecycle | Providers, Agents |

The **Runtime** is your infrastructure - things that start when the service starts and shutdown when it stops. The Database connection pool. The HTTP server. The logger.

**Domain Systems** are your business logic - they use Runtime systems but don't have their own lifecycle. A Providers repository doesn't "start" or "shutdown". It just needs a database connection to do its job.

```go
// Runtime owns the infrastructure
type Runtime struct {
    Lifecycle  *lifecycle.Coordinator
    Logger     logger.System
    Database   database.System
    Pagination pagination.Config
}

// Domain owns the business logic
type Domain struct {
    Providers providers.System
    // Agents agents.System  (coming soon)
}

// Service ties them together
type Service struct {
    runtime *Runtime
    domain  *Domain
    server  server.System
}
```

This separation means you always know where to look. Infrastructure problem? Check Runtime. Business logic issue? It's in Domain.

---

## Cold Start, Hot Start

Every system has two phases:

**Cold Start** (`New*()` functions) - Build the entire dependency graph. Create all the structs. Wire everything together. But nothing is running yet. No goroutines, no open connections, no listening sockets.

**Hot Start** (`Start()` methods) - Flip the switch. Open the database connection. Start the HTTP server. Begin listening for signals.

```go
// Cold Start - build the graph
svc, err := NewService(cfg)
if err != nil {
    log.Fatal(err)
}

// Hot Start - activate everything
if err := svc.Start(); err != nil {
    log.Fatal(err)
}
```

Why does this matter? Because when something goes wrong during initialization, you know exactly where you are. Config validation failed? You're still cold - nothing to clean up. Database connection failed during startup? You know what's been activated and what hasn't.

---

## The Lifecycle Coordinator

Shutdown is where most services get messy. You've got goroutines everywhere, database connections that need closing, HTTP requests in flight...

We centralized all of this into a **Lifecycle Coordinator**:

```go
// Subsystems register what they need
lc.OnStartup(func() {
    // Ping database, warm caches, etc.
})

lc.OnShutdown(func() {
    <-lc.Context().Done()  // Wait for shutdown signal
    // Clean up gracefully
})
```

The coordinator handles the orchestration. When `Shutdown()` is called:

1. Context is cancelled
2. All `OnShutdown` hooks fire (in goroutines)
3. We wait for everything to complete (with timeout)
4. Service exits cleanly

No more scattered cleanup code. No more "did I remember to close that connection?"

---

## State Flows Down, Never Up

This is the rule that makes everything else work.

**Wrong:**
```go
type Handler struct {
    service *Service  // Handler stores reference to parent
}

func (h *Handler) Process() {
    sys := h.service.Providers()  // Reaching UP to get state
}
```

**Right:**
```go
func HandleCreate(w http.ResponseWriter, r *http.Request,
                  system providers.System, logger *slog.Logger) {
    // State is passed DOWN at the call site
}
```

When state flows down, you can trace any request from entry point to database and see exactly what's available at each step. No hidden dependencies. No magic.

---

## Configuration That Actually Makes Sense

Three layers, atomic replacement at each level:

```
Environment Variables (highest priority)
    ↓ replaces (not merges)
config.dev.toml / config.prod.toml
    ↓ replaces (not merges)
config.toml (base)
```

The key insight: **values replace, they don't merge**. If you set `CORS_ORIGINS` in your environment, it completely replaces whatever was in the TOML file. No weird partial merging. No surprises.

```bash
# This replaces the TOML array entirely
CORS_ORIGINS="http://prod.example.com,http://api.example.com"
```

Configuration is also **ephemeral** - it's used during initialization, then discarded. Systems don't hold references to config structs. They extract what they need during construction and that's it.

---

## The Query Builder

We built a three-layer query system that turned out to be one of the cleanest parts of the architecture:

**Layer 1: Projection Map** - Define your table structure once
```go
var projection = query.NewProjectionMap("public", "providers", "p").
    Project("id", "Id").
    Project("name", "Name").
    Project("config", "Config")
```

**Layer 2: Builder** - Fluent API for conditions
```go
qb := query.NewBuilder(projection, "Name").
    WhereSearch(page.Search, "Name").
    OrderBy(page.SortBy, page.Descending)
```

**Layer 3: Execute** - Generate SQL and run it
```go
countSQL, countArgs := qb.BuildCount()
pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
```

The builder automatically ignores nil/empty filter values. So you can always apply all your filters and let the builder figure out which ones matter.

---

## What's Coming: Providers and Agents

The first domain system being built is **Providers** - LLM provider configurations (Ollama, Azure AI, etc.). It demonstrates the full pattern:

```
internal/providers/
├── provider.go      # State structures
├── errors.go        # Domain errors
├── projection.go    # Query structure
├── system.go        # Interface
├── repository.go    # Implementation
├── handlers.go      # HTTP handlers
└── routes.go        # Route group
```

Each provider configuration gets validated against the actual go-agents library during creation - if your config wouldn't work, you find out immediately, not when you try to use it.

Next up: **Agents** - which combine a provider reference with model configuration. Then workflows. Then the real fun begins.

---

## The Philosophy

A few principles that guided everything:

1. **Explicit over implicit** - If you need something, pass it in. Don't reach for it.

2. **Composition over inheritance** - Go doesn't have inheritance anyway, but the point stands. Small, focused systems that compose together.

3. **Errors are values** - Each package defines its errors in `errors.go`. They wrap with context. They map cleanly to HTTP status codes.

4. **Tests are black-box** - All tests use `package foo_test` and only test the public interface. If it's not exported, it's not tested directly.

5. **No magic** - When you read the code, what you see is what happens.

---

## Try It

```bash
# Start PostgreSQL
docker compose -f compose/postgres.yml up -d

# Run migrations
go run ./cmd/migrate -dsn "postgres://..." -up

# Start the service
go run ./cmd/service

# Check health
curl http://localhost:8080/healthz

# Check readiness (waits for database)
curl http://localhost:8080/readyz
```

The readiness endpoint (`/readyz`) returns `NOT READY` until all startup hooks complete. Once the database ping succeeds, it flips to `READY`. Simple, but useful.

---

## What's Next

This is still pre-release. The foundation is solid, but we're building towards something bigger - a platform where you can design, test, and deploy agentic AI workflows. Document classification. Data extraction. Multi-agent collaboration with confidence scoring.

But that's the vision. Right now, it's a really clean Go web service that handles configuration, lifecycle, database access, and HTTP routing in a way that actually makes sense.

Want to dig deeper? Check out `ARCHITECTURE.md` for the full technical spec. Or just read the code - it's designed to be readable.

---

*Built on Go 1.25, PostgreSQL 17, and a healthy skepticism of frameworks.*
