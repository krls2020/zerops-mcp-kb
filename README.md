# Zerops Knowledge API

Simple, semantic search API for Zerops knowledge base inspired by Context7's architecture.

## Architecture

This API follows a simple two-endpoint design inspired by Context7's MCP approach:

1. **Search** - Find relevant knowledge using simple text queries
2. **Get** - Retrieve full knowledge content by semantic ID

### Design Philosophy

- **Simplicity**: Just two endpoints - search and get
- **Semantic IDs**: Human-readable identifiers like `service/nodejs` or `recipe/laravel`
- **Fast**: In-memory index built at startup (~160 items)
- **Flexible Search**: Space or comma separated terms
- **Context-Aware**: Smart relevance scoring based on multiple factors

### How It Works

```
User Query → Search Endpoint → Relevance Scoring → Return Summaries
                                                          ↓
                                                    Get Full Content
                                                          ↓
                                                  Knowledge Endpoint
```

The API loads all knowledge files at startup into an in-memory index with semantic IDs. When searching, it:
1. Parses the query into terms (comma or space separated)
2. Scores each knowledge item based on ID matches, framework/language matches, and content frequency
3. Returns top results with summaries and tags
4. Full content can be retrieved using the semantic ID

## API Endpoints

### 1. Search - `/api/v1/search`

Simple text search with comma or space separation.

**Request:**
```json
POST /api/v1/search
{
  "query": "nodejs, postgresql",
  "limit": 10
}
```

**Response:**
```json
{
  "query": "nodejs, postgresql",
  "results": [
    {
      "id": "recipe/nodejs",
      "name": "Nodejs",
      "summary": "Node.js with optimized production setup",
      "type": "recipe",
      "tags": ["recipe", "nodejs", "javascript"],
      "score": 2.5
    },
    {
      "id": "service/nodejs",
      "name": "Nodejs",
      "summary": "Node.js runtime for JavaScript and TypeScript applications",
      "type": "service",
      "tags": ["service", "nodejs", "runtime"],
      "score": 2.0
    },
    {
      "id": "service/postgresql",
      "name": "Postgresql",
      "summary": "PostgreSQL relational database service",
      "type": "service",
      "tags": ["service", "postgresql", "database"],
      "score": 1.8
    }
  ],
  "count": 3
}
```

### 2. Get Knowledge - `/api/v1/knowledge/{id}`

Get full knowledge content by semantic ID.

**Request:**
```
GET /api/v1/knowledge/recipe/laravel-jetstream
```

**Response:**
```json
{
  "id": "recipe/laravel-jetstream",
  "name": "laravel-jetstream",
  "type": "recipe",
  "content": {
    // Full JSON content from the knowledge file
    "framework": "Laravel",
    "language": "PHP",
    "services": [...],
    "zeropsYml": {...},
    "bestPractices": [...]
  }
}
```

## Semantic ID Structure

All knowledge has semantic IDs in format: `{type}/{name}`

### Types:
- `service/` - Zerops services (nodejs, postgresql, mariadb, etc.)
- `recipe/` - Deployment recipes (laravel, django, nextjs, etc.)
- `patterns/` - Deployment patterns
- `runtimes/` - Runtime configurations
- `nginx/` - Nginx configurations

### Examples:
```
services/nodejs
services/postgresql
recipe/laravel-jetstream
recipe/django
recipe/nextjs
patterns/nextjs-full-stack
runtimes/go
nginx/wordpress
```

## Search Examples

### Simple searches:
```json
// Find Node.js related content
{"query": "nodejs"}

// Find Laravel recipes
{"query": "laravel"}

// Find database services
{"query": "database"}

// Find PostgreSQL and Node.js
{"query": "nodejs postgresql"}
{"query": "nodejs, postgresql"}  // Same result
```

### Common search patterns:
```json
// Find deployment recipes for specific framework
{"query": "nextjs deployment"}
{"query": "laravel recipe"}

// Find services for a tech stack
{"query": "nodejs postgresql redis"}

// Find Python-related content
{"query": "python django"}

// Find container/Docker related
{"query": "docker container"}

// Find specific service versions
{"query": "postgresql 16"}
{"query": "nodejs 20"}

// Find recipes with specific features
{"query": "mailpit email"}
{"query": "github runner"}
```

### Search behavior:
- Space or comma separated terms
- Case insensitive
- Searches in: ID, name, description, framework, language, tags
- Returns best matches sorted by relevance score
- Score based on:
  - ID matches (highest weight)
  - Framework/language matches (high weight)
  - Content frequency (medium weight)

## Run

```bash
go run main.go
PORT=3000 go run main.go
```

## Live Demo

The API is running at: **https://kbapi-167b-8080.prg1.zerops.app/**

## Test Examples

### Using the Live Demo Server

```bash
# Search for Node.js
curl -X POST https://kbapi-167b-8080.prg1.zerops.app/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "nodejs"}'

# Search for Laravel and PostgreSQL
curl -X POST https://kbapi-167b-8080.prg1.zerops.app/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "laravel, postgresql"}'

# Get specific Laravel recipe
curl https://kbapi-167b-8080.prg1.zerops.app/api/v1/knowledge/recipe/laravel-jetstream

# Get Node.js service info
curl https://kbapi-167b-8080.prg1.zerops.app/api/v1/knowledge/services/nodejs

# Health check
curl https://kbapi-167b-8080.prg1.zerops.app/health
```

### Local Development

```bash
# Search for Node.js
curl -X POST localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "nodejs"}'

# Search for Laravel and PostgreSQL
curl -X POST localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "laravel, postgresql"}'

# Get specific Laravel recipe
curl localhost:8080/api/v1/knowledge/recipe/laravel-jetstream

# Get Node.js service info
curl localhost:8080/api/v1/knowledge/services/nodejs

# Health check
curl localhost:8080/health
```

## Implementation Details

### Code Structure
- **Single file**: `main.go` (~300 lines)
- **No dependencies**: Pure Go standard library
- **Embedded data**: Knowledge files compiled into binary

### Scoring Algorithm
```go
// Relevance scoring weights:
- ID match: 2.0 points
- Framework/Language match: 1.0/0.8 points  
- Content frequency: 0.1 points per occurrence
- Score normalized by number of search terms
```

### Index Building
At startup, the API:
1. Walks through all JSON files in `knowledge/data/`
2. Generates semantic IDs from file paths
3. Cleans up prefixes (e.g., `recipe-` → `recipe/`)
4. Stores in memory map for O(1) lookups

## Deployment

### Zerops Deployment
The API is deployed on Zerops as a Go service:
- **Project**: zerops-kb-api
- **Service**: kbapi (go@1)
- **URL**: https://kbapi-zerops-kb-api.prg1.zerops.app

### Configuration
- Port configured via `PORT` environment variable (default: 8080)
- No database required - all data embedded
- Stateless - can scale horizontally

## MCP Integration

This API is designed for integration with MCP servers (like Context7):

1. MCP server receives user query
2. Calls `/api/v1/search` to find relevant knowledge
3. Selects best match based on relevance scores
4. Calls `/api/v1/knowledge/{id}` for full content
5. Injects knowledge into LLM context

Example MCP flow:
```javascript
// 1. Search for relevant knowledge
const results = await fetch('/api/v1/search', {
  body: JSON.stringify({query: "nodejs deployment"})
});

// 2. Get top result's full content
const knowledge = await fetch(`/api/v1/knowledge/${results[0].id}`);

// 3. Add to LLM context
context.add(knowledge.content);
```