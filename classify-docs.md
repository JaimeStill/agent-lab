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
      "document_id": "2cc7c735-6963-410d-8a0e-b852f605e1e5",
      "agent_id": "e75dedb2-9054-41fa-b97f-2f05f76a3e6d"
    },
    "token": "a40368d4ef4b4fd59eb330026e747ce4"
  }'
```
