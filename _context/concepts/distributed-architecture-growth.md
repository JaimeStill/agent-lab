# Distributed Architecture Growth

Considerations for scaling the stdlib-native HTTP architecture to support distributed client synchronization and cluster orchestration.

## Current Architecture Assessment

The existing infrastructure uses Go's standard library (`net/http`) with custom abstractions:

- **Module system**: Isolated handler groups with path prefixes and independent middleware chains
- **Route groups**: Hierarchical route organization with OpenAPI integration
- **Middleware stack**: Composable `func(http.Handler) http.Handler` pattern
- **Lifecycle coordination**: Graceful startup/shutdown via `lifecycle.Coordinator`
- **SSE streaming**: Working server-sent events for real-time server-to-client push

This architecture handles the hard coordination problems internally, providing full control over infrastructure behavior.

## Framework Comparison: Fiber vs Standard Library

### Fiber Advantages

| Feature | Benefit |
|---------|---------|
| Built-in middleware | Rate limiting, compression, CSRF, caching out of the box |
| Request binding/validation | One-liner with struct tags via `c.Bind().Body(user)` |
| Per-route middleware | Apply middleware at any group level |
| Fasthttp performance | Zero-allocation patterns, lower per-request overhead |

### Standard Library Advantages

| Feature | Benefit |
|---------|---------|
| No framework lock-in | Handlers use portable `http.ResponseWriter` / `*http.Request` |
| Go 1.22+ routing | `"METHOD /path"` syntax is sufficiently expressive |
| Stability | `net/http` is backwards-compatible, no breaking upgrades |
| Full control | Debug and modify infrastructure without upstream coordination |

### Decision

The switching cost outweighs Fiber's incremental benefits. The existing custom infrastructure is mature and addresses the same problems Fiber solves. Migration would require rewriting every handler, middleware, and test.

## WebSocket Integration Path

Go's standard library does not include WebSocket support. Third-party libraries required:

| Library | Characteristics |
|---------|-----------------|
| `gorilla/websocket` | Battle-tested, widely used, simple API |
| `nhooyr.io/websocket` | Context-aware, modern patterns, ~2000 LOC with no transitive deps |

### Integration Complexity: Low

WebSocket handlers are standard `http.HandlerFunc` signatures—they integrate directly with the existing module system. Required custom code:

1. **Connection registry**: Track active connections for broadcasting
2. **Message protocol**: Define JSON message envelope types
3. **Graceful shutdown**: Close connections during lifecycle teardown

### SSE vs WebSocket

SSE is already implemented and sufficient for server-to-client push. WebSockets add:

- Bidirectional communication (client sends structured messages)
- Binary message support
- Lower overhead for high-frequency messaging

Introduce WebSockets when bidirectional communication becomes a requirement.

## Distributed System Architecture

Future state supporting multi-node deployment with client synchronization:

```
                                    ┌─────────────────┐
                                    │  Message Broker │
                                    │  (NATS / Redis) │
                                    └────────┬────────┘
                                             │ pub/sub
                 ┌───────────────────────────┼───────────────────────────┐
                 │                           │                           │
          ┌──────▼──────┐             ┌──────▼──────┐             ┌──────▼──────┐
          │   Node A    │             │   Node B    │             │   Node C    │
          │  ┌───────┐  │             │  ┌───────┐  │             │  ┌───────┐  │
          │  │WS Hub │  │             │  │WS Hub │  │             │  │WS Hub │  │
          │  └───┬───┘  │             │  └───┬───┘  │             │  └───┬───┘  │
          └──────┼──────┘             └──────┼──────┘             └──────┼──────┘
                 │                           │                           │
              Clients                     Clients                     Clients
```

### Message Broker Options

| Broker | Fit |
|--------|-----|
| **NATS** | Lightweight, Go-native, supports pub/sub and request/reply |
| **Redis Pub/Sub** | Simple if Redis already in stack, no persistence |
| **RabbitMQ** | Complex routing needs, heavier operationally |
| **Kafka** | Event sourcing, replay requirements, high throughput |

NATS is the strongest candidate for Go-native systems requiring sync and orchestration without event sourcing.

### Integration Pattern

Define a `pubsub.System` interface in the infrastructure layer:

```go
type System interface {
    Publish(ctx context.Context, subject string, data []byte) error
    Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error)
    RequestReply(ctx context.Context, subject string, data []byte) ([]byte, error)
}
```

The WebSocket hub subscribes to broker subjects and fans out to locally-connected clients. This isolates the broker choice behind an interface, allowing single-node development without a broker.

## Dependency Philosophy

### Guiding Principles

1. **Stdlib-first**: Use standard library unless capability is genuinely unavailable
2. **Narrow interfaces**: When dependencies are unavoidable, wrap them behind owned interfaces
3. **Vendor-ready**: Prefer small, focused libraries that could be vendored if abandoned
4. **Control over convenience**: Accept higher initial effort for long-term maintainability

### Dependency Risk Profile

| Type | Risk | Examples |
|------|------|----------|
| Framework | High | Fiber, Gin, Echo—touches everything, upgrade churn |
| Protocol library | Low | `websocket`, `jwt`—narrow interface, wrap and isolate |
| Driver | Low | `pgx`, `go-sql-driver`—stdlib interface abstracts |

### Unavoidable Dependencies

Some capabilities require external packages:

- Database drivers (stdlib `database/sql` abstracts the driver)
- WebSocket protocol upgrade
- Cryptographic protocols (JWT, OAuth token handling)

These are acceptable when:
- The interface is narrow and stable
- The library has minimal transitive dependencies
- Vendoring is feasible as a fallback

## Scaling Assessment

The current stdlib-native architecture scales well to emerging requirements:

| Requirement | Path Forward |
|-------------|--------------|
| WebSocket sync | Add hub + `nhooyr.io/websocket`, fits existing lifecycle |
| Multi-node broadcast | Add `pubsub.System` interface, implement with NATS |
| Per-route middleware | Extend `Route` struct to include middleware slice |
| Request validation | Generic decode helper + `go-playground/validator` |

The custom infrastructure (~500-800 lines) is:
- Readable in an afternoon
- Testable in isolation
- Modifiable without upstream coordination

The maintenance burden of owning this code is lower than the coordination cost of framework dependency management.
