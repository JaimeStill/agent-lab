# Recursive Modules

## Concept

Modules could serve as mount points for sub-modules, creating a hierarchical tree structure. A "proxy module" would establish shared infrastructure (middleware, services, configuration) that flows down to its child modules.

## Example Architecture

```
Server (native: /healthz, /readyz)
├── /api       → API Module
├── /app       → App Module
├── /scalar    → Scalar Module
└── /apps      → Apps Proxy Module (shared middleware, auth, etc.)
    ├── /apps/admin   → Admin App (inherits /apps middleware)
    └── /apps/portal  → Portal App (inherits /apps middleware)
```

## Design Considerations

### Middleware Inheritance

Should a request to `/apps/admin/dashboard` run through:
- `/apps` middleware first, then `/apps/admin` middleware? (chained)
- Or just the most specific module's middleware? (isolated)

Chained middleware enables shared concerns (auth, logging, rate limiting) at the proxy level.

### Proxy vs. Handler

Can a proxy module have its own handler (e.g., landing page at `/apps/`), or is it purely a middleware container for children?

Options:
1. Proxy modules have no handler - only middleware and children
2. Proxy modules can optionally have a handler for their root path
3. Proxy modules ARE regular modules that happen to have children mounted

### API Shape Options

**Option A: Flat mounting with implicit hierarchy**
```go
router.Mount(appsModule)        // /apps - proxy with shared middleware
router.Mount(adminModule)       // /apps/admin - inherits from /apps
router.Mount(portalModule)      // /apps/portal - inherits from /apps
```
Router infers hierarchy from prefix relationships.

**Option B: Explicit tree**
```go
appsModule.Mount(adminModule)   // /apps mounts /admin internally
appsModule.Mount(portalModule)
router.Mount(appsModule)        // Only root modules mounted to router
```
Explicit parent-child relationships.

**Option C: Builder pattern**
```go
router.Mount("/apps", func(apps *Module) {
    apps.Use(sharedMiddleware)
    apps.Mount("/admin", adminModule)
    apps.Mount("/portal", portalModule)
})
```
Nested configuration.

### Implementation Complexity

- Router needs to handle longest-prefix matching (already implemented via sort)
- Middleware chaining requires tracking parent-child relationships
- Request path rewriting becomes more complex with multiple levels
- Testing hierarchical middleware is more involved

## Use Cases

1. **Multi-tenant applications** - `/tenants/{tenant}/` proxy with tenant-specific middleware
2. **Application suites** - `/apps/` proxy for multiple related web applications
3. **API versioning alternative** - `/api/v1/`, `/api/v2/` as separate modules (though internal routing may be cleaner)

## Current Decision

For mt06, modules are constrained to single-level sub-paths (`/api`, `/app`, `/scalar`). Multiple distinct apps are mounted at root level rather than nested.

This concept can be revisited when a concrete use case emerges that benefits from hierarchical mounting.
