# go-agents v0.3.0: Provider/Model Decoupling

**Context**: This artifact was generated during agent-lab Session 1c milestone review. Use this to initialize a Claude Code session in go-agents to execute the architecture adjustment.

## Problem Statement

The current go-agents configuration hierarchy tightly couples Provider and Model:

```
agent: { client: { provider: { model: {} } } }
```

This coupling limits standalone provider functionality in consuming applications (like agent-lab). To validate a provider configuration, consumers must include ModelConfig even though:

1. Providers don't validate model configuration during creation
2. Providers don't use model configuration during initialization
3. Model is simply passed through to downstream processing

This prevents agent-lab from managing providers as independent entities.

## Analysis Summary

**Current Flow:**
1. `Client.New()` receives `ClientConfig` containing `ProviderConfig`
2. `providers.Create()` is called with `ProviderConfig`
3. Provider factory (e.g., `NewOllama`) requires `ProviderConfig.Model` to be non-nil
4. Factory calls `types.FromConfig(c.Model)` immediately
5. Provider stores model internally

**Key Finding:** The coupling is **structural, not functional**. Provider factories don't validate model configuration - they just pass it through. The model is only used later during request preparation.

**Evidence:**
- `ollama.go`: Doesn't validate model, just calls `types.FromConfig(c.Model)`
- `azure.go`: Validates provider options (deployment, auth_type, token, api_version) but not model
- `BaseProvider`: Stores `*types.Model` but doesn't validate it

## Proposed Architecture

Flatten the configuration hierarchy so provider and model are peers:

```
Current:
agent: {
  client: {
    provider: {
      model: {}  // Nested inside provider
    }
  }
}

Proposed:
agent: {
  client: {}    // Connection settings (timeout, retry, pool)
  provider: {}  // Provider config (name, base_url, options)
  model: {}     // Model config (name, capabilities)
}
```

**Key Changes:**
1. `ProviderConfig` no longer contains `ModelConfig`
2. Provider factories validate provider-specific config only
3. Model is wired at Client or Agent level, not Provider level
4. Providers become reusable across different models

## Files to Modify

### Configuration Layer (`pkg/config/`)

**`provider.go`** - Remove ModelConfig field:
```go
// Before
type ProviderConfig struct {
    Name    string
    BaseURL string
    Options map[string]any
    Model   *ModelConfig  // Remove this
}

// After
type ProviderConfig struct {
    Name    string
    BaseURL string
    Options map[string]any
}
```

**`client.go`** - Add ModelConfig as peer:
```go
type ClientConfig struct {
    Timeout        Duration
    Retry          *RetryConfig
    ConnPoolSize   int
    ConnTimeout    Duration
    Provider       *ProviderConfig
    Model          *ModelConfig  // Add here (peer to Provider)
}
```

**`agent.go`** - Or add at agent level if preferred:
```go
type AgentConfig struct {
    Name         string
    SystemPrompt string
    Client       *ClientConfig
    Model        *ModelConfig  // Alternative: at agent level
}
```

### Provider Layer (`pkg/providers/`)

**`base.go`** - Remove model from BaseProvider:
```go
// Before
type BaseProvider struct {
    name    string
    baseURL string
    model   *types.Model
}

// After
type BaseProvider struct {
    name    string
    baseURL string
}
```

**`provider.go`** - Remove Model() from interface:
```go
// Before
type Provider interface {
    Name() string
    BaseURL() string
    Model() *types.Model
    // ... protocol methods
}

// After
type Provider interface {
    Name() string
    BaseURL() string
    // Model removed - lives at Client/Agent level
    // ... protocol methods
}
```

**`ollama.go`** - Simplify factory:
```go
func NewOllama(c *config.ProviderConfig) (Provider, error) {
    baseURL := formatURL(c.BaseURL)
    return &OllamaProvider{
        BaseProvider: NewBaseProvider(c.Name, baseURL),
        options:      c.Options,
    }, nil
}
```

**`azure.go`** - Simplify factory (keep provider validation):
```go
func NewAzure(c *config.ProviderConfig) (Provider, error) {
    // Validate provider-specific options
    deployment := getOption[string](c.Options, "deployment")
    if deployment == "" {
        return nil, fmt.Errorf("azure provider requires 'deployment' option")
    }
    // ... other provider validation

    return &AzureProvider{
        BaseProvider: NewBaseProvider(c.Name, baseURL),
        // ... provider fields
    }, nil
}
```

### Client Layer (`pkg/client/`)

**`client.go`** - Wire provider and model separately:
```go
func New(cfg *config.ClientConfig) (Client, error) {
    provider, err := providers.Create(cfg.Provider)
    if err != nil {
        return nil, fmt.Errorf("create provider: %w", err)
    }

    model := types.FromConfig(cfg.Model)

    return &client{
        provider: provider,
        model:    model,  // Stored at client level
        config:   cfg,
    }, nil
}
```

### Mock Layer (`pkg/mock/`)

**`provider.go`** - Update mock to match new interface.

## Migration Path

This is a **breaking change** requiring v0.3.0:

1. **Config structure changes** - `ProviderConfig.Model` removed
2. **Provider interface changes** - `Model()` method removed
3. **Factory signature unchanged** - Still takes `*ProviderConfig`
4. **Client responsibility** - Wires provider + model together

### Migration for Consumers

Before:
```go
providerCfg := &config.ProviderConfig{
    Name:    "ollama",
    BaseURL: "http://localhost:11434",
    Model: &config.ModelConfig{
        Name: "llama3.2",
    },
}
```

After:
```go
providerCfg := &config.ProviderConfig{
    Name:    "ollama",
    BaseURL: "http://localhost:11434",
}

modelCfg := &config.ModelConfig{
    Name: "llama3.2",
}

clientCfg := &config.ClientConfig{
    Provider: providerCfg,
    Model:    modelCfg,
}
```

## Testing Strategy

### Provider Tests (`tests/providers/`)

Update all provider tests to:
1. Remove ModelConfig from ProviderConfig in test fixtures
2. Verify providers create successfully without model
3. Verify provider validation still works (Azure options, etc.)

### Client Tests (`tests/client/`)

Update client tests to:
1. Create provider and model separately
2. Verify client wires them together correctly
3. Verify model is accessible from client

### Agent Tests (`tests/agent/`)

Update agent tests to:
1. Use new config structure
2. Verify end-to-end flow still works

### Integration Tests

Run existing integration tests with updated config to verify:
1. Ollama provider works with decoupled model
2. Azure provider works with decoupled model
3. All protocol methods (chat, vision, tools) work correctly

## Benefits

1. **Standalone Provider Management**: agent-lab can validate and store providers independently
2. **Provider Reuse**: Same provider can be used with different models
3. **Clearer Separation**: Provider = connection config, Model = inference config
4. **Simpler Validation**: Provider factories only validate provider-specific settings
5. **Flexibility**: Model can live at Client or Agent level based on use case

## Implementation Order

1. Update `pkg/config/` - Move ModelConfig out of ProviderConfig
2. Update `pkg/providers/` - Remove model from providers
3. Update `pkg/client/` - Wire provider and model at client level
4. Update `pkg/mock/` - Match new interface
5. Update `pkg/agent/` - Adjust if model moves to agent level
6. Update all tests
7. Update documentation and examples
