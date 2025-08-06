# Zerops Knowledge API

Semantic search API for Zerops knowledge base with smart relevance scoring.

**Live Demo:** https://kbapi-167b-8080.prg1.zerops.app/

## Quick Start

```bash
# Search for knowledge
curl -X POST https://kbapi-167b-8080.prg1.zerops.app/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "nodejs postgresql"}'

# Get specific knowledge
curl https://kbapi-167b-8080.prg1.zerops.app/api/v1/knowledge/recipe/laravel-jetstream
```

## API Endpoints

### Search: `POST /api/v1/search`
Find relevant knowledge using text queries. Returns best matches based on framework, language, and content relevance.

```json
{
  "query": "python flask postgresql",
  "limit": 10
}
```

### Get: `GET /api/v1/knowledge/{id}`
Retrieve full knowledge by semantic ID like `services/nodejs` or `recipe/flask`.

## Features

- **Smart Scoring**: Prioritizes documents matching ALL search terms
- **Framework-Aware**: Flask searches return Flask recipes first, not generic Python
- **Fast**: In-memory index with <10ms response time
- **Simple**: Just two endpoints - search and get

## Semantic IDs

- `services/` - Zerops services (nodejs, postgresql)
- `recipe/` - Deployment recipes (flask, django, laravel-jetstream)
- `patterns/` - Deployment patterns (flask, nextjs-full-stack)

## Run Locally

```bash
go run main.go
# Or with custom port
PORT=3000 go run main.go
```

## Deploy to Zerops

```bash
zcli push --projectId YOUR_PROJECT_ID kbapi
```

Repository: https://github.com/krls2020/zerops-mcp-kb