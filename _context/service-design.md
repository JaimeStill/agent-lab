# agent-lab Service Design

**Status**: Future design patterns and conceptual systems
**Scope**: Patterns not yet implemented or under consideration

This document captures agent-lab specific architectural designs for future milestones. For implemented patterns, see [ARCHITECTURE.md](../ARCHITECTURE.md). For general architectural philosophy, see [web-service-architecture.md](./web-service-architecture.md).

## Milestone 1: Implemented & Documented

**All Milestone 1 patterns have been implemented and documented in ARCHITECTURE.md:**

- ✅ **Foundation Infrastructure** - Config, Logger, Routes, Middleware, Server, Lifecycle
- ✅ **Database & Query Infrastructure** - Database system, Query builder, Pagination, Repository helpers
- ✅ **Domain Systems** - Runtime/Domain separation, Providers system, Agents system
- ✅ **Domain Infrastructure Patterns** - Handler struct, Filters, Scanner, Projection, Error mapping
- ✅ **HTTP Patterns** - Request/response handling, Error responses, SSE streaming

**Pattern Evolution Notes:**

Milestone 1 implementations evolved from initial designs in this document:
- **Handler Pattern**: Evolved from pure functions to Handler struct with Routes() method
- **Transaction Management**: Abstracted into pkg/repository.WithTx[T]
- **Query Building**: Implemented ProjectionMap + Builder pattern
- **Error Handling**: Domain errors with MapHTTPStatus pattern
- **Agents Schema**: Agents embed full provider config (no foreign key to providers table)
- **Search/Pagination**: Unified into PageRequest with filters applied via domain Filters.Apply()

See ARCHITECTURE.md for current implementation details.

## Milestone 2+: Future Patterns

*This section will be populated as future milestone patterns are designed.*

### Blob Storage Abstraction (Milestone 2)

**Not yet designed - To be added during Milestone 2 planning**

Considerations:
- Filesystem vs Azure Blob Storage implementation
- Interface abstraction for storage backends
- Transaction coordination between DB and blob storage
- Cleanup strategies for orphaned blobs

### Long-Running Systems (Milestone 3)

**Not yet designed - To be added during Milestone 3 planning**

Considerations:
- Execution queue system (channel-based)
- Worker pool system (goroutine management)
- Event bus system (pub/sub messaging)
- Integration with lifecycle coordinator
- Graceful shutdown with in-flight work

### Workflow Orchestration Patterns (Milestone 5)

**Not yet designed - To be added during Milestone 5 planning**

Considerations:
- go-agents-orchestration integration
- Workflow state management
- Multi-stage execution patterns
- Confidence scoring architecture
- Result aggregation strategies

---

## Historical Note

This document previously contained detailed designs for Provider and Agent systems, database schemas, error handling, testing strategies, and deployment patterns. All of these have been implemented in Milestone 1 with pattern evolution (Handler struct, repository helpers, query builder, etc.).

See ARCHITECTURE.md for current implementation patterns.
