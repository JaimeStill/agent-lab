# Agent Containers

A deployment model where containerized environments serve as capability-bounded execution contexts for autonomous AI reasoning.

## Core Premise

Agent containers are general-purpose autonomous agents operating within container boundaries. The container image defines the complete capability surface - tools, skills, permissions, and organizational interfaces. External systems feed context and objectives; the agent reasons autonomously about how to accomplish the work.

This parallels how developers work. A developer doesn't "call a Git service" - Git is installed, configured, and available. The developer reasons about what to do, then executes commands in their environment. Agent containers provide the same model for AI systems.

**Key insight**: These are not specialized task processors (document classifier, image analyzer). They are autonomous agents that can do anything within their capability boundary - query data systems, correlate findings, generate reports, produce visualizations, interface with organizational tools.

## The Claude Code Parallel

The mental model is Claude Code deployed as a service:

| Aspect | Claude Code | Agent Container |
|--------|-------------|-----------------|
| Execution context | User's machine | Container boundary |
| Context source | Conversation + local filesystem | External systems + injected data |
| Tool availability | Installed on user's machine | Packaged in container image |
| Organizational access | User's credentials/access | Configured interfaces |
| Interaction model | Interactive conversation | Work submission API |

Both share the fundamental pattern: an autonomous agent with a rich toolchain, reasoning about how to accomplish objectives using available capabilities.

## Architecture

```
┌─────────────────────────────────────────────────┐
│ Agent Container                                 │
├─────────────────────────────────────────────────┤
│  Primary Agent                                  │
│  ├── System prompt (autonomous behavior)        │
│  └── External interface (work submission)       │
├─────────────────────────────────────────────────┤
│  Toolchain                                      │
│  ├── File operations (read, write, edit)        │
│  ├── Code execution (Python, Go, Node, etc.)    │
│  ├── Data processing (SQL, pandas, etc.)        │
│  ├── Document/media (ImageMagick, FFmpeg)       │
│  └── Network/API (curl, configured clients)     │
├─────────────────────────────────────────────────┤
│  Organizational Interfaces                      │
│  ├── Pre-configured at build time               │
│  └── Injected at runtime (org-specific)         │
├─────────────────────────────────────────────────┤
│  Skills                                         │
│  ├── Compositional patterns + tools             │
│  ├── Model targeting (skill-specific)           │
│  └── Skill-specific system prompts              │
├─────────────────────────────────────────────────┤
│  Sub-Agents                                     │
│  ├── Delegated autonomous reasoning             │
│  └── Model targeting (task-appropriate)         │
├─────────────────────────────────────────────────┤
│  Agent Runtime                                  │
│  ├── Multi-model routing                        │
│  ├── Sub-agent invocation                       │
│  ├── State management                           │
│  └── Checkpoint persistence                     │
├─────────────────────────────────────────────────┤
│  Uniform API                                    │
│  └── Work submission interface                  │
└─────────────────────────────────────────────────┘
```

### Primary Agent

The primary agent is the container's interface to the external world. Like Claude Code, it has an extensive but optimized system prompt that programs its toolchain and expected behaviors. This prompt:

- Defines the agent's role and capabilities
- Provides guidance on tool selection and composition
- Establishes behavioral boundaries and preferences
- Documents available organizational interfaces
- Describes when to delegate to skills or sub-agents

The primary agent receives work submissions, reasons about approach, and orchestrates execution - but it's not necessarily the only model involved in completing the work.

### Toolchain

The container packages a rich set of capabilities - not minimal task-specific tools, but a comprehensive environment for autonomous work:

- **File/code operations**: Read, write, edit, execute - the fundamentals
- **Language runtimes**: Python, Go, Node, etc. for computation
- **Data processing**: SQL clients, data manipulation libraries
- **Document/media**: Processing, transformation, generation
- **Network/API**: HTTP clients, configured service interfaces

### Skills

Skills are compositional units the agent can chain - not just "tools with documentation" but patterns that combine tools, produce intermediate state, and signal logical next steps. A skill might internally use multiple tools and represent a reusable approach to a class of problems.

**Model targeting**: Skills can specify which model executes them. A data validation skill might target a fast, cost-efficient model (haiku-class), while a complex analysis skill targets a more capable model (sonnet/opus-class). Each skill carries its own system prompt optimized for that specific task.

```yaml
skill: validate-data-format
model: haiku
system_prompt: |
  You validate data against schema definitions.
  Return structured validation results.
tools: [jq, jsonschema]

skill: analyze-trends
model: sonnet
system_prompt: |
  You perform statistical analysis and identify patterns.
  Explain findings with supporting evidence.
tools: [python3, pandas, matplotlib]
```

This enables optimization across multiple dimensions:
- **Performance**: Fast models for simple, well-defined tasks
- **Cost**: Cheaper models where reasoning complexity is low
- **Accuracy**: More capable models for nuanced decisions
- **Latency**: Right-sized models for time-sensitive operations

Skills load based on context relevance rather than all-at-once, managing context window constraints while preserving reasoning quality.

### Sub-Agents

Beyond skills, containers can invoke sub-agents for work requiring delegated autonomous reasoning. While skills are structured patterns with defined tools, sub-agents are autonomous reasoners that receive objectives and context, then determine their own approach.

```
Primary Agent receives complex objective
  → Decomposes into sub-problems
  → Delegates sub-problem to sub-agent (possibly different model)
  → Sub-agent reasons autonomously within its scope
  → Returns results to primary agent
  → Primary agent synthesizes and continues
```

Sub-agents enable:
- **Parallel reasoning**: Multiple sub-agents work concurrently on independent sub-problems
- **Specialized reasoning**: Sub-agents with domain-specific prompts and model selection
- **Bounded context**: Each sub-agent operates with focused context rather than full history
- **Cost optimization**: Simple sub-tasks use efficient models; complex sub-tasks use capable models

The primary agent acts as orchestrator, delegating appropriately while maintaining overall coherence.

### Organizational Interfaces

This is where containers become organization-specific. Interfaces can be:

**Pre-configured**: Baked into the image for standard integrations (common APIs, databases, services).

**Runtime-injected**: Loaded at deployment with organization-specific instructions:
- "Here's how to query our data warehouse"
- "Here's how to access our CRM"
- "Here's how to publish to our reporting system"
- "Here's our data schema and business terminology"

General-purpose container images become organization-specific through configuration injection, not image rebuilds.

## Trust Model

### Container Boundary = Capability Boundary = Trust Boundary

The container image freezes the capability surface at build time. An agent cannot acquire new tools or permissions at runtime. This provides:

**Auditability**: Security review happens once per image version. Certified capabilities don't change between certification and deployment.

**Predictability**: Integration tests validate the complete capability surface. No runtime surprises from dynamic capability discovery.

**Isolation**: Containers cannot affect each other or the host beyond their defined interfaces. The agent's autonomy is bounded by the tools present.

### Capability Immutability

Skills declare what tools they use. Tools declare what system access they need. The container build process validates that all dependencies resolve and permissions are sufficient.

A container built without network access simply cannot include skills that require it. The capability absence is structural, not policy-enforced.

### Credentials

Containers receive credentials through environment injection at deployment, never baked into images. The agent runtime exposes credentials to skills through a controlled interface with scoping per-skill.

## Uniform API

Every agent container exposes the same interface regardless of internal capabilities:

```
POST   /execute          # Submit work, receive streaming results
POST   /execute/sync     # Submit work, receive complete result
GET    /capabilities     # Enumerate available skills/tools
GET    /runs/{id}        # Query execution state
DELETE /runs/{id}        # Cancel execution
```

The `/execute` payload is **work submission**, not task invocation:

```json
{
  "objective": "Analyze Q4 sales trends and prepare executive summary",
  "context": {
    "data_sources": ["sales_db", "crm", "market_data"],
    "output_format": "pdf_report",
    "audience": "executive",
    "additional_context": "Focus on regional variance and YoY comparison"
  },
  "options": {
    "timeout": "30m",
    "checkpoint": true
  }
}
```

The agent interprets the objective based on available capabilities. The caller specifies *what* they want accomplished, not *how* to accomplish it. The agent reasons about tool selection, skill composition, and execution strategy.

### Capability Manifest

Containers publish capabilities as a machine-readable manifest:

```yaml
name: analytics-agent
version: 1.0.0

capabilities:
  skills:
    - data-analysis
    - report-generation
    - visualization
    - data-correlation

  tools:
    - python3
    - postgresql-client
    - imagemagick
    - pandoc

  interfaces:
    - type: database
      name: analytics_db
    - type: api
      name: crm_api
    - type: storage
      name: report_output

limits:
  max_runtime: 1h
  max_output_size: 100MB
```

This enables orchestration systems to understand container capabilities without understanding internals.

## Composition Patterns

### Container Orchestration

Complex work may span multiple containers when capabilities are distributed:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Data Agent     │────▶│ Analysis Agent  │────▶│  Report Agent   │
│  (extraction)   │     │  (processing)   │     │  (generation)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

An orchestration layer routes work between containers based on capability matching. Each container operates independently with its own agent runtime.

### Nested Invocation

Containers can invoke other containers when their capabilities are insufficient:

```
Analytics Agent receives request requiring specialized ML inference
  → Recognizes: "I need model inference I don't have locally"
  → Discovers: ML Agent has required capability
  → Delegates: POST ml-agent/execute with inference request
  → Receives: Inference results
  → Continues: Incorporates results into analysis
```

The invoking agent reasons about delegation just as it reasons about local tool selection.

## Relationship to Explicit Orchestration

Agent containers don't replace explicit workflow orchestration (like go-agents-orchestration). They complement it:

**Explicit orchestration** (workflow graphs) excels at:
- Well-understood processes with deterministic paths
- Compliance workflows requiring provable execution sequences
- High-volume processing where predictability enables optimization
- Debugging scenarios where reproducibility matters

**Agent containers** excel at:
- Exploratory work where optimal path is unknown
- Heterogeneous inputs requiring content-dependent routing
- Complex objectives requiring dynamic tool composition
- Human-in-the-loop workflows adapting to feedback

The hybrid model: explicit orchestration can invoke agent containers for steps requiring autonomous reasoning, while agent containers can follow structured patterns when appropriate.

## Future Optimization Directions

These are not near-term goals but architectural considerations for later development:

### GPU Compute Scheduling

LLM inference has predictable compute characteristics. Containers could declare compute profiles enabling capacity planning:

- Per-request GPU memory and compute requirements
- Thread/concurrency limits
- Burst allowances for complex reasoning chains

This would enable deterministic cost attribution and capacity-aware request routing.

### Edge Deployment

Consumer hardware (128GB unified memory systems) now runs 70B parameter models at conversational speeds. This enables:

- Air-gapped deployments for sensitive workloads
- Field deployment for disconnected operations
- Privacy-sensitive processing that cannot leave premises

Containers would use different compute profiles (optimizing for latency on sequential requests vs. throughput on concurrent requests).

## Open Questions

**Skill Conflict Resolution**: When multiple skills could apply, how does the agent choose? Confidence scoring? Explicit priority? Context-dependent selection?

**Cross-Container State**: How do containers share intermediate state during orchestrated workflows? Reference passing? Shared storage? State tokens?

**Version Compatibility**: How do orchestration systems handle capability changes across container versions? Semantic versioning on capabilities?

**Failure Boundaries**: When a delegated container fails, how does the invoking agent reason about recovery? Retry? Alternative path? Escalation?

**Context Window Management**: With extensive system prompts and multiple skills, how do you balance capability richness against reasoning quality as context fills?

**Organizational Interface Validation**: How do you validate that runtime-injected interfaces actually work before the agent attempts to use them?

**Model Routing Decisions**: How does the primary agent decide when to use skill-specific models vs. sub-agents vs. handling directly? Static rules in skill definitions? Dynamic reasoning about task complexity?

**Sub-Agent Coordination**: When multiple sub-agents work in parallel, how do you handle dependencies between their outputs? Barrier synchronization? Streaming partial results?

**Cost Budgeting**: With multi-model execution, how do you track and limit cost across a single work submission? Per-skill budgets? Overall execution budget with model selection adapting?

## Relationship to agent-lab

Agent containers represent a potential evolution path for agent-lab's architecture:

| agent-lab Concept | Agent Container Equivalent |
|-------------------|---------------------------|
| Workflow Graph | Explicit orchestration layer (retained) |
| Workflow Runtime | Agent Runtime |
| Observer | Execution trace capture |
| CheckpointStore | Checkpoint persistence |
| Profile + Stages | Capability configuration |
| Domain Systems | Organizational interfaces |

The primary addition is autonomous reasoning about tool/skill composition within structured boundaries. Explicit workflows remain valuable; agent containers add flexibility for work that doesn't fit predetermined paths.

## Summary

Agent containers package autonomous AI reasoning with execution capabilities into deployable, auditable units. The container boundary defines what an agent *can* do. The system prompt and skills define *how*. The agent reasons about *when* and *why*.

This treats agent deployment like application deployment: versioned artifacts with known capabilities, tested before release, monitored in production. Autonomous reasoning operates within engineering constraints rather than around them.
