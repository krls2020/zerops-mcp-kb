# Zerops Knowledge API

Semantic search API for Zerops knowledge base inspired by Context7's architecture.

**Live Demo:** https://kbapi-167b-8080.prg1.zerops.app/

## Architecture

Simple two-endpoint design following MCP patterns:

1. **Search** - Find relevant knowledge using text queries
2. **Get** - Retrieve full content by semantic ID

### Design Philosophy

- **Simplicity**: Just two endpoints - search and get
- **Semantic IDs**: Human-readable identifiers like `services/nodejs` or `recipe/flask`
- **Framework-Aware**: Flask searches return Flask recipes first, not generic Python

### How It Works

```
User Query → Search Endpoint → Relevance Scoring → Return Summaries
                                                          ↓
                                                    Get Full Content
                                                          ↓
                                                  Knowledge Endpoint
```

The API loads all knowledge at startup into memory with semantic IDs. When searching:
1. Parses query into terms (space/comma separated)
2. Scores items based on framework, language, ID matches
3. Returns ranked results with summaries
4. Full content retrievable via semantic ID

## API Endpoints

### Search: `POST /api/v1/search`

```json
{
  "query": "python flask postgresql",
  "limit": 10
}
```

Returns ranked results with IDs, summaries, and scores.

### Get: `GET /api/v1/knowledge/{id}`

```bash
GET /api/v1/knowledge/patterns/flask
GET /api/v1/knowledge/services/nodejs
GET /api/v1/knowledge/recipe/laravel-jetstream
```

Returns full JSON content of the knowledge item.

## Semantic ID Structure

Format: `{type}/{name}`

- `services/` - Zerops services (nodejs, postgresql, mariadb)
- `recipe/` - Deployment recipes (flask, django, laravel-jetstream)
- `patterns/` - Deployment patterns
- `runtimes/` - Runtime configurations
- `nginx/` - Nginx configurations

## Quick Examples

```bash
# Search for Flask stack
curl -X POST https://kbapi-167b-8080.prg1.zerops.app/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "python flask postgresql"}'

# Get specific knowledge
curl https://kbapi-167b-8080.prg1.zerops.app/api/v1/knowledge/patterns/flask
```

### Search Patterns

```json
{"query": "flask"}                      // Framework search
{"query": "python postgresql flask"}    // Multi-term (matches ALL)
{"query": "nodejs postgresql redis"}    // Tech stack
{"query": "postgresql 16"}             // Version-specific
```

## Local Development

```bash
go run main.go
PORT=3000 go run main.go
```
