# Execution Snippets

Quick reference for terminal execution.

## Agents

### Ollama Chat

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ollama-agent",
    "config": {
      "name": "ollama-agent",
      "system_prompt": "You are an expert software architect specializing in cloud native systems design",
      "client": {
        "timeout": "24s",
        "retry": {
          "max_retries": 3,
          "initial_backoff": "1s",
          "max_backoff": "30s",
          "backoff_multiplier": 2.0,
          "jitter": true
        },
        "connection_pool_size": 10,
        "connection_timeout": "9s"
      },
      "provider": {
        "name": "ollama",
        "base_url": "http://localhost:11434"
      },
      "model": {
        "name": "llama3.2:3b",
        "capabilities": {
          "chat": {
            "max_tokens": 4096,
            "temperature": 0.7,
            "top_p": 0.95
          }
        }
      }
    }
  }'
```

### Ollama Vision

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "vision-agent",
    "config": {
      "name": "vision-agent",
      "client": {
        "timeout": "24s",
        "retry": {
          "max_retries": 3,
          "initial_backoff": "1s",
          "max_backoff": "30s",
          "backoff_multiplier": 2.0,
          "jitter": true
        },
        "connection_pool_size": 10,
        "connection_timeout": "9s"
      },
      "provider": {
        "name": "ollama",
        "base_url": "http://localhost:11434"
      },
      "model": {
        "name": "gemma3:4b",
        "capabilities": {
          "vision": {
            "max_tokens": 4096,
            "temperature": 0.7,
            "vision_options": {
              "detail": "auto"
            }
          }
        }
      }
    }
  }'
```

### Ollama Embeddings

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "embeddings-agent",
    "config": {
      "name": "embeddings-agent",
      "client": {
        "timeout": "24s",
        "retry": {
          "max_retries": 3,
          "initial_backoff": "1s",
          "max_backoff": "30s",
          "backoff_multiplier": 2.0,
          "jitter": true
        },
        "connection_pool_size": 10,
        "connection_timeout": "6s"
      },
      "provider": {
        "name": "ollama",
        "base_url": "http://localhost:11434"
      },
      "model": {
        "name": "embeddinggemma:300m",
        "capabilities": {
          "embeddings": {
            "dimensions": 768
          }
        }
      }
    }
  }'
```

### Azure OpenAI (API Key)

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "azure-key-agent",
    "config": {
      "name": "azure-key-agent",
      "system_prompt": "You are an expert software architect specializing in cloud native systems design",
      "client": {
        "timeout": "24s",
        "retry": {
          "max_retries": 3,
          "initial_backoff": "1s",
          "max_backoff": "30s",
          "backoff_multiplier": 2.0,
          "jitter": true
        },
        "connection_pool_size": 10,
        "connection_timeout": "9s"
      },
      "provider": {
        "name": "azure",
        "base_url": "https://your-resource.openai.azure.com/openai",
        "options": {
          "deployment": "your-deployment",
          "api_version": "2025-01-01-preview",
          "auth_type": "api_key",
          "token": "token"
        }
      },
      "model": {
        "name": "gpt-4o",
        "capabilities": {
          "chat": {
            "max_completion_tokens": 4096
          }
        }
      }
    }
  }'
```

### Azure OpenAI (Entra ID)

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "azure-entra-agent",
    "config": {
      "name": "azure-entra-agent",
      "system_prompt": "You are an expert software architect specializing in cloud native systems design",
      "client": {
        "timeout": "24s",
        "retry": {
          "max_retries": 3,
          "initial_backoff": "1s",
          "max_backoff": "30s",
          "backoff_multiplier": 2.0,
          "jitter": true
        },
        "connection_pool_size": 10,
        "connection_timeout": "9s"
      },
      "provider": {
        "name": "azure",
        "base_url": "https://your-resource.openai.azure.com/openai",
        "options": {
          "deployment": "your-deployment",
          "api_version": "2025-01-01-preview",
          "auth_type": "bearer",
          "token": "token"
        }
      },
      "model": {
        "name": "gpt-4o",
        "capabilities": {
          "chat": {
            "max_completion_tokens": 4096
          }
        }
      }
    }
  }'
```

### Azure GPT-5-mini (Vision)

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gpt-5-mini",
    "config": {
      "name": "gpt-5-mini",
      "provider": {
        "name": "azure",
        "base_url": "https://go-agents-platform.openai.azure.com/openai",
        "options": {
          "deployment": "gpt-5-mini",
          "api_version": "2025-01-01-preview",
          "auth_type": "api_key",
          "token": "token"
        }
      },
      "model": {
        "name": "gpt-5-mini",
        "capabilities": {
          "chat": {
            "max_completion_tokens": 4096
          },
          "vision": {
            "max_completion_tokens": 4096,
            "vision_options": {
              "detail": "high"
            }
          }
        }
      }
    }
  }'
```

## Agent Execution

### Chat

```bash
curl -X POST http://localhost:8080/api/agents/{id}/chat \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What is cloud native architecture?"}'
```

### Chat Stream (SSE)

```bash
curl -N -X POST http://localhost:8080/api/agents/{id}/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What is cloud native architecture?"}'
```

### Vision

```bash
curl -X POST http://localhost:8080/api/agents/{id}/vision \
  -F "prompt=What do you see in this image?" \
  -F "images=@/path/to/image.png"
```

### Vision (Multiple Images)

```bash
curl -X POST http://localhost:8080/api/agents/{id}/vision \
  -F "prompt=Compare these images" \
  -F "images=@/path/to/image1.png" \
  -F "images=@/path/to/image2.png"
```

### Vision Stream (SSE)

```bash
curl -N -X POST http://localhost:8080/api/agents/{id}/vision/stream \
  -F "prompt=Describe this image" \
  -F "images=@/path/to/image.png"
```

### Tools

```bash
curl -X POST http://localhost:8080/api/agents/{id}/tools \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What is the weather in New York?",
    "tools": [
      {
        "name": "get_weather",
        "description": "Get the current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string", "description": "The city name"}
          },
          "required": ["location"]
        }
      }
    ]
  }'
```

### Embed

```bash
curl -X POST http://localhost:8080/api/agents/{id}/embed \
  -H "Content-Type: application/json" \
  -d '{"input": "Cloud native architecture emphasizes microservices and containerization."}'
```

### With Azure Token

```bash
curl -N -X POST http://localhost:8080/api/agents/{id}/chat/stream \
  -H "Content-Type: application/json" \
  -d "{\"prompt\": \"Hello\", \"token\": \"$AZURE_KEY\"}"
```

## Workflows

### Execute classify-docs

```bash
curl -X POST http://localhost:8080/api/workflows/classify-docs/execute \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "document_id": "1b760735-a0da-4c45-9868-e311e3c4117d",
      "agent_id": "2b0c0844-e267-42a6-b45f-e08f08d1de5f"
    },
    "token": "<api-key>"
  }'
```

### Execute classify-docs (with Profile)

```bash
curl -X POST http://localhost:8080/api/workflows/classify-docs/execute \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "document_id": "<document-uuid>",
      "agent_id": "<agent-uuid>",
      "profile_id": "<profile-uuid>"
    },
    "token": "<api-key>"
  }'
```

### List Workflow Runs

```bash
curl http://localhost:8080/api/workflows/runs
```

### Get Run Details

```bash
curl http://localhost:8080/api/workflows/runs/{id}
```

### Get Run Stages

```bash
curl http://localhost:8080/api/workflows/runs/{id}/stages
```

### Get Run Decisions

```bash
curl http://localhost:8080/api/workflows/runs/{id}/decisions
```

## Documents

### Upload

```bash
curl -X POST http://localhost:8080/api/documents \
  -F "file=@/path/to/document.pdf"
```

### List

```bash
curl http://localhost:8080/api/documents
```

### Render Pages

```bash
curl -X POST http://localhost:8080/api/images/{documentId}/render \
  -H "Content-Type: application/json" \
  -d '{"pages": "1-5", "dpi": 150, "format": "png"}'
```

### Get Image Binary

```bash
curl http://localhost:8080/api/images/{id}/data -o image.png
```
