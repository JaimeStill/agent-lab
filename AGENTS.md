# Agents API Guide

This guide covers the Agents API endpoints for creating, managing, and executing AI agents.

## Agent Configurations

### Ollama Chat Agent

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
          },
          "tools": {
            "max_tokens": 4096,
            "temperature": 0.7,
            "tool_choice": "auto"
          }
        }
      }
    }
  }'
```

### Ollama Vision Agent

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
          "chat": {
            "max_tokens": 4096,
            "temperature": 0.7,
            "top_p": 0.95
          },
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

### Ollama Embeddings Agent

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

### Azure OpenAI Agent (API Key)

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

### Azure OpenAI Agent (Entra ID Bearer Token)

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

## CRUD Endpoints

### Create Agent

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-agent",
    "config": { ... }
  }'
```

### List Agents

```bash
curl http://localhost:8080/api/agents
```

With pagination and filtering:

```bash
curl "http://localhost:8080/api/agents?page=1&pageSize=10&name=ollama"
```

### Get Agent by ID

```bash
curl http://localhost:8080/api/agents/{id}
```

### Update Agent

```bash
curl -X PUT http://localhost:8080/api/agents/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "updated-name",
    "config": { ... }
  }'
```

### Delete Agent

```bash
curl -X DELETE http://localhost:8080/api/agents/{id}
```

### Search Agents (POST)

```bash
curl -X POST http://localhost:8080/api/agents/search \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "pageSize": 10,
    "search": "azure",
    "sort": [{"field": "name", "direction": "asc"}]
  }'
```

## Execution Endpoints

### Chat

```bash
curl -X POST http://localhost:8080/api/agents/{id}/chat \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What is cloud native architecture?"
  }'
```

### Chat Stream (SSE)

```bash
curl -N -X POST http://localhost:8080/api/agents/{id}/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What is cloud native architecture?"
  }'
```

### Vision (File Upload)

```bash
curl -X POST http://localhost:8080/api/agents/{id}/vision \
  -F "prompt=What do you see in this image?" \
  -F "images=@/path/to/image.png"
```

Multiple images:

```bash
curl -X POST http://localhost:8080/api/agents/{id}/vision \
  -F "prompt=Compare these images" \
  -F "images=@/path/to/image1.png" \
  -F "images=@/path/to/image2.png"
```

### Vision Stream (File Upload, SSE)

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
            "location": {
              "type": "string",
              "description": "The city name"
            }
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
  -d '{
    "input": "Cloud native architecture emphasizes microservices and containerization."
  }'
```

## Token Authentication (Azure)

For Azure agents, pass the authentication token at request time. The token overrides the placeholder value stored in the agent config.

### API Key Authentication

```bash
curl -N -X POST http://localhost:8080/api/agents/{id}/chat/stream \
  -H "Content-Type: application/json" \
  -d "{\"prompt\": \"Hello\", \"token\": \"$AZURE_KEY\"}"
```

### Entra ID Bearer Token

```bash
curl -N -X POST http://localhost:8080/api/agents/{id}/chat/stream \
  -H "Content-Type: application/json" \
  -d "{\"prompt\": \"Hello\", \"token\": \"$AZURE_TOKEN\"}"
```

### Vision with Token

```bash
curl -X POST http://localhost:8080/api/agents/{id}/vision \
  -F "prompt=What is in this image?" \
  -F "images=@/path/to/image.png" \
  -F "token=$AZURE_KEY"
```

### Tools with Token

```bash
curl -X POST http://localhost:8080/api/agents/{id}/tools \
  -H "Content-Type: application/json" \
  -d "{
    \"prompt\": \"What is the weather?\",
    \"tools\": [...],
    \"token\": \"$AZURE_KEY\"
  }"
```

### Embed with Token

```bash
curl -X POST http://localhost:8080/api/agents/{id}/embed \
  -H "Content-Type: application/json" \
  -d "{\"input\": \"Text to embed\", \"token\": \"$AZURE_KEY\"}"
```
