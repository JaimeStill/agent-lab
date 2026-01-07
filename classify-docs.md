# Classify Docs Test Run

```sh
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

```sh
curl -X POST http://localhost:8080/api/workflows/classify-docs/execute/stream \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "document_id": "<document-id>",
      "agent_id": "<agent-id>"
    },
    "token": "<api-key>"
  }'
```
