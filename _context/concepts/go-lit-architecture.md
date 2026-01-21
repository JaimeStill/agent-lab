# Go + Lit Web Application Architecture

## Overview

A server-rendered shell architecture where Go owns data and routing concerns while Lit web components own presentation and interaction entirely.

**Boundary Constraint**: This separation is a hard architectural rule, not a guideline.

- Go never renders view-specific markup—only data contracts (JSON APIs) and the static shell
- Lit components never assume server behavior beyond documented API responses
- The shell's only job is bootstrapping the runtime and providing the mount point

## Server Architecture (Go)

### Responsibilities

- HTTP routing (API endpoints + single catch-all for `/app/*`)
- Authentication and authorization
- Data serialization (JSON API responses)
- Serving the static `app.html` shell and bundled assets

### Shell Rendering

A single `app.html` template serves all `/app/*` routes. The template is static—Go has no awareness of which view will render. It provides:

- Document structure and meta tags
- Script tags loading the Lit component bundle
- A mount point element for the client router

## Client Architecture (Lit + TypeScript)

### Build Tooling

- **Bun**: Runtime and package manager
- **Vite**: Build orchestration (uses esbuild internally for dev, Rollup for production)
- **TypeScript**: All client code

Vite provides HMR support for Lit components out of the box.

### Router Design

**Model**: Static mapping for optimized tree-shaking.

The router:

1. Reads `location.pathname` and extracts the view segment (`/app/workflows` → `workflows`)
2. Looks up the corresponding component tag name from a static map
3. Creates and mounts the component to the content container
4. Intercepts internal link clicks for `pushState` navigation
5. Listens on `popstate` for browser back/forward

**URL Parsing**: The router handles all route parsing concerns and provides convenience functions for accessing URL data. Components receive parsed data—they never parse URLs directly. This keeps components portable and testable.

**Route Parameters**: For routes like `/app/workflows/:id`, the router parses params and passes them as attributes on mount.

## Component Hierarchy

Three distinct component types with clear responsibility boundaries:

| Type | Service Awareness | State Ownership | Data Flow |
|------|-------------------|-----------------|-----------|
| **View** | Initializes and provides | Owns service lifecycle | Router state → services |
| **Stateful** | Consumes via context | Subscribes to service signals | Services ↔ events |
| **Stateless** | None | None | Attributes → events |

### View Components

- Top-level components that the router mounts
- Responsible for initializing service-level state
- Use `@provide()` to make services available to descendants
- The **only** component type that provides services
- Receive initial router state and use it to initialize dependencies

### Stateful Components

- View segments that encapsulate sub-sections of a view
- Use `@consume()` to access services via context
- Serve as state isolation barriers
- Capture and handle events from children by invoking service methods
- Events do not propagate beyond the stateful boundary

### Stateless Components

- Lowest-level component primitives
- Behave like native HTML elements: building blocks
- Receive state directly via attributes
- Emit events in response to user interactions
- No concept of service dependencies or context participation
- Pure and highly reusable

## Directory-to-Component Mapping

The directory structure reflects the component hierarchy directly:

| Directory | Component Type | Responsibility |
|-----------|----------------|----------------|
| **views/** | View components | Initialize services, provide context, receive router state |
| **components/** | Stateful components | Consume services, handle events, coordinate state |
| **elements/** | Stateless components | Receive attributes, emit events, pure rendering |

Each layer imports only from layers below. Within a domain, views compose components, components compose elements. Elements have no upward dependencies.

## State Management

### Cold/Hot Start Paradigm

Treat router and service state initialization the same way Go configurations work.

**Cold Start**: Router state flows downward as constructor/attribute data. Each component initializes its own dependencies before rendering children. The tree hydrates depth-first; parents gate child mounting until their own initialization completes.

**Hot Path**: Signals indicate readiness. A parent exposing an initialized signal lets children subscribe and defer setup until dependencies are available. Ongoing state changes flow through the same signal graph.

### Signal Ownership

- **Component-level signals**: Encapsulated state, scoped to component instance trees
- **Service-level signals**: Orchestrated state access, shared across consumers

This provides a consistent debugging model: when state behaves unexpectedly, ask "is this component-owned or service-owned?" and the answer indicates where to look.

## Service Architecture

### Structure

Services are pure TypeScript modules. They handle:

- API calls
- State machines
- Validation logic
- Business rules

Services do not depend on Lit. Components consume them via import or context.

### Interface-Based Contracts

Services are defined by interfaces, not implementations:

```typescript
// interfaces.ts
export interface WorkflowService {
  load(id: string): Promise<Workflow>;
  save(workflow: Workflow): Promise<void>;
}
```

Context objects are typed with the interface:

```typescript
// contexts.ts
import { createContext } from '@lit/context';
import type { WorkflowService } from './interfaces';

export const workflowServiceContext = createContext<WorkflowService>('workflow-service');
```

View components provide concrete implementations. Stateful components consume the interface type with no visibility into the backing implementation.

### Context Protocol

Uses Lit's `@provide()` / `@consume()` decorators:

- View components `@provide()` individual services (not a container)
- Stateful components `@consume()` the interfaces they depend on
- The consumed contexts serve as a readable manifest of component dependencies
- Stateless components have no context participation

Providing individual services rather than a container makes dependencies explicit at both ends and simplifies testing.

## Event Flow

Clear propagation boundaries based on component type:

1. **Stateless components** emit events in response to user interaction
2. **Stateful components** catch events and handle them by invoking service methods
3. Events do not need to propagate beyond the stateful boundary

This avoids the "event relay" problem where intermediate components pass events upward without acting on them.

## Testing Strategy

Testing tiers align with component types:

| Component Type | Test Approach |
|----------------|---------------|
| **Stateless** | Attribute inputs, event assertions, no mocking required |
| **Stateful** | Mock service implementations that satisfy the interface |
| **View** | Service initialization, container registration, integration tests |

The interface-based service design enables test isolation—provide mock implementations without infrastructure.

## Data Loading

View components receive initial state from the router as attributes. Each view component can:

- Fetch additional data in `connectedCallback`
- Initialize services with the router-provided state
- Signal readiness to children once initialization completes

For navigation within a view (e.g., `/app/workflows/123` → `/app/workflows/456`), the router can update attributes on the existing component rather than remounting, triggering Lit's reactive update cycle.

## Directory Structure (Suggested)

Organized by domain, not atomic layer. Each domain contains everything needed to operate within that domain.

```
src/
├── app.ts                 # Entry point, component registration
├── router/
│   ├── router.ts          # Navigation, path parsing, mounting
│   └── routes.ts          # Static path → component mapping
├── shared/                # Cross-domain infrastructure
│   ├── elements/          # Shared stateless primitives (ui-button, ui-icon)
│   ├── types/
│   └── utils/
├── domains/
│   ├── workflows/
│   │   ├── views/         # View components (router targets)
│   │   ├── components/    # Stateful components
│   │   ├── elements/      # Domain-specific stateless components
│   │   ├── interfaces.ts  # Service and type contracts
│   │   ├── types.ts       # Domain-specific types
│   │   ├── context.ts     # Service context definitions
│   │   └── ...            # Any other domain infrastructure
│   ├── agents/
│   │   ├── views/
│   │   ├── components/
│   │   ├── elements/
│   │   ├── interfaces.ts
│   │   └── ...
│   └── [domain]/
└── services/
    ├── workflows/
    │   ├── interfaces.ts  # Service interface definitions
    │   ├── impl.ts        # Concrete implementation
    │   └── ...            # Factories, singletons, utilities
    ├── agents/
    │   └── ...
    └── [domain]/
```

### Domain Structure

Each domain under `domains/` can contain:

- **views/**: View components (router targets)
- **components/**: Stateful components
- **elements/**: Domain-specific stateless components
- **interfaces.ts**: Contracts for domain types and services
- **types.ts**: Domain-specific type definitions
- **context.ts**: Lit context definitions for service provision
- Any additional infrastructure: classes, singletons, factories, utilities

Services under `services/` mirror domain boundaries but contain no component infrastructure—purely business logic, API clients, and state management.

### Cross-Domain Code

`shared/` contains infrastructure used across multiple domains:

- Common stateless elements (design system primitives)
- Shared type definitions
- Utility functions

Domain code imports from `shared/`. Domains should not import from each other directly—if two domains need to communicate, that coordination belongs in a service or at the view level.

## Key Principles

1. **Hard boundary**: Go owns data, Lit owns presentation
2. **Explicit dependencies**: Context consumption declares what a component needs
3. **Interface contracts**: Services are consumed by interface, not implementation
4. **Event ceilings**: Stateful components are the handling boundary
5. **Cold start flows down**: Initialization is deterministic and depth-first
6. **Signals for readiness**: Hot path uses reactive primitives
