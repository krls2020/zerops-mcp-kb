package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

//go:embed knowledge/data
var data embed.FS

type SearchResult struct {
	ID          string  `json:"id"`          // Semantic ID like "service/nodejs"
	Name        string  `json:"name"`        
	Summary     string  `json:"summary"`     
	Type        string  `json:"type"`        // service, pattern, runtime, recipe
	Tags        []string `json:"tags"`       
	Score       float64 `json:"score"`       
}

type Knowledge struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

// Build index at startup for better performance
var knowledgeIndex = make(map[string]*Knowledge)

func main() {
	buildIndex()
	
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/v1/search", handleSearch)
	http.HandleFunc("/api/v1/knowledge/", handleGetKnowledge)
	http.HandleFunc("/health", handleHealth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting on :%s", port)
	log.Printf("Indexed %d knowledge items", len(knowledgeIndex))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func buildIndex() {
	fs.WalkDir(data, "knowledge/data", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		content, err := data.ReadFile(path)
		if err != nil {
			return nil
		}

		// Generate semantic ID
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			return nil
		}
		
		dir := parts[len(parts)-2]
		name := strings.TrimSuffix(parts[len(parts)-1], ".json")
		
		// Clean up recipe prefixes
		if strings.HasPrefix(name, "recipe-") {
			name = strings.TrimPrefix(name, "recipe-")
			dir = "recipe"
		}
		
		id := fmt.Sprintf("%s/%s", dir, name)
		
		knowledgeIndex[id] = &Knowledge{
			ID:      id,
			Name:    name,
			Type:    dir,
			Content: content,
		}
		
		return nil
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// Only handle exact root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Zerops Knowledge Base API</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            background: white;
            border-radius: 10px;
            padding: 40px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
        }
        h1 {
            color: #764ba2;
            border-bottom: 3px solid #667eea;
            padding-bottom: 10px;
        }
        h2 {
            color: #667eea;
            margin-top: 30px;
        }
        .endpoint {
            background: #f8f9fa;
            border-left: 4px solid #667eea;
            padding: 15px;
            margin: 20px 0;
            border-radius: 5px;
        }
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 3px;
            font-weight: bold;
            font-size: 12px;
            margin-right: 10px;
        }
        .get { background: #28a745; color: white; }
        .post { background: #007bff; color: white; }
        code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        pre {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
        }
        .stats {
            display: flex;
            gap: 20px;
            margin: 20px 0;
        }
        .stat {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px 20px;
            border-radius: 8px;
            flex: 1;
            text-align: center;
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
        }
        .stat-label {
            font-size: 0.9em;
            opacity: 0.9;
        }
        a {
            color: #667eea;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .example {
            margin: 15px 0;
        }
        .try-button {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 10px 20px;
            border-radius: 5px;
            text-decoration: none;
            margin: 10px 5px;
            transition: transform 0.2s;
        }
        .try-button:hover {
            transform: translateY(-2px);
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Zerops Knowledge Base API</h1>
        <p>Simple, semantic search API for the Zerops platform knowledge base. Fast, in-memory search with semantic IDs.</p>
        
        <div class="stats">
            <div class="stat">
                <div class="stat-value">` + fmt.Sprintf("%d", len(knowledgeIndex)) + `</div>
                <div class="stat-label">Knowledge Items</div>
            </div>
            <div class="stat">
                <div class="stat-value">2</div>
                <div class="stat-label">API Endpoints</div>
            </div>
            <div class="stat">
                <div class="stat-value">&lt;10ms</div>
                <div class="stat-label">Response Time</div>
            </div>
        </div>

        <h2>📚 API Endpoints</h2>
        
        <div class="endpoint">
            <span class="method post">POST</span>
            <code>/api/v1/search</code>
            <p>Search for knowledge using simple text queries. Supports comma or space separated terms.</p>
            <div class="example">
                <strong>Example Request:</strong>
                <pre>{
  "query": "nodejs postgresql",
  "limit": 10
}</pre>
            </div>
        </div>

        <div class="endpoint">
            <span class="method get">GET</span>
            <code>/api/v1/knowledge/{id}</code>
            <p>Get full knowledge content by semantic ID (e.g., <code>services/nodejs</code>, <code>recipe/laravel-jetstream</code>)</p>
            <div class="example">
                <strong>Example:</strong> <code>/api/v1/knowledge/recipe/laravel-jetstream</code>
            </div>
        </div>

        <div class="endpoint">
            <span class="method get">GET</span>
            <code>/health</code>
            <p>Health check endpoint for monitoring</p>
        </div>

        <h2>🎯 Try It Out</h2>
        <p>Quick examples to get you started:</p>
        
        <a href="/api/v1/knowledge/services/nodejs" class="try-button">View Node.js Service</a>
        <a href="/api/v1/knowledge/recipe/laravel-jetstream" class="try-button">View Laravel Recipe</a>
        <a href="/api/v1/knowledge/services/postgresql" class="try-button">View PostgreSQL Service</a>
        <a href="/health" class="try-button">Health Check</a>

        <h2>💻 Example Usage</h2>
        <pre>curl -X POST https://kbapi-167b-8080.prg1.zerops.app/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "nodejs postgresql"}'</pre>

        <h2>📖 Documentation</h2>
        <p>For complete documentation and more examples, visit the <a href="https://github.com/krls2020/zerops-mcp-kb" target="_blank">GitHub Repository</a></p>
        
        <h2>🔗 Semantic ID Structure</h2>
        <p>All knowledge items use semantic IDs in the format: <code>{type}/{name}</code></p>
        <ul>
            <li><code>service/</code> - Zerops services (nodejs, postgresql, mariadb, etc.)</li>
            <li><code>recipe/</code> - Deployment recipes (laravel, django, nextjs, etc.)</li>
            <li><code>patterns/</code> - Deployment patterns</li>
            <li><code>runtimes/</code> - Runtime configurations</li>
            <li><code>nginx/</code> - Nginx configurations</li>
        </ul>
    </div>
</body>
</html>`
	
	fmt.Fprint(w, html)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if req.Limit <= 0 || req.Limit > 20 {
		req.Limit = 10
	}

	// Parse query - support comma separation
	terms := parseQuery(req.Query)
	results := search(terms, req.Limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
	})
}

func handleGetKnowledge(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/v1/knowledge/service/nodejs
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/knowledge/")
	
	knowledge, exists := knowledgeIndex[id]
	if !exists {
		http.Error(w, "Knowledge not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(knowledge)
}

func parseQuery(query string) []string {
	// Split by comma or space, clean up
	query = strings.ToLower(query)
	terms := []string{}
	
	for _, part := range strings.FieldsFunc(query, func(r rune) bool {
		return r == ',' || r == ' '
	}) {
		term := strings.TrimSpace(part)
		if term != "" {
			terms = append(terms, term)
		}
	}
	
	return terms
}

func search(terms []string, limit int) []SearchResult {
	var results []SearchResult
	
	for id, knowledge := range knowledgeIndex {
		score := calculateScore(knowledge, terms)
		if score > 0 {
			// Extract summary from content
			var obj map[string]interface{}
			json.Unmarshal(knowledge.Content, &obj)
			
			summary := ""
			if desc, ok := obj["description"].(string); ok {
				summary = desc
				if len(summary) > 200 {
					summary = summary[:197] + "..."
				}
			}
			
			// Extract tags
			tags := extractTags(obj, knowledge.Type)
			
			results = append(results, SearchResult{
				ID:      id,
				Name:    formatName(knowledge.Name),
				Summary: summary,
				Type:    knowledge.Type,
				Tags:    tags,
				Score:   score,
			})
		}
	}
	
	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	if len(results) > limit {
		results = results[:limit]
	}
	
	return results
}

func calculateScore(knowledge *Knowledge, terms []string) float64 {
	if len(terms) == 0 {
		return 0.1 // Show all with low score if no terms
	}
	
	contentStr := strings.ToLower(string(knowledge.Content))
	idLower := strings.ToLower(knowledge.ID)
	score := 0.0
	
	for _, term := range terms {
		// Exact ID match
		if strings.Contains(idLower, term) {
			score += 2.0
		}
		
		// Check in content
		count := strings.Count(contentStr, term)
		if count > 0 {
			score += float64(count) * 0.1
		}
		
		// Bonus for framework/language matches
		if strings.Contains(contentStr, `"framework":"`) && strings.Contains(contentStr, term) {
			score += 1.0
		}
		if strings.Contains(contentStr, `"language":"`) && strings.Contains(contentStr, term) {
			score += 0.8
		}
	}
	
	// Normalize score
	if score > 10.0 {
		score = 10.0
	}
	
	return score / float64(len(terms))
}

func extractTags(obj map[string]interface{}, knowledgeType string) []string {
	tags := []string{knowledgeType}
	
	// Extract existing tags
	if tagList, ok := obj["tags"].([]interface{}); ok {
		for _, tag := range tagList {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}
	
	// Add framework as tag
	if framework, ok := obj["framework"].(string); ok {
		tags = append(tags, strings.ToLower(framework))
	}
	
	// Add language as tag
	if language, ok := obj["language"].(string); ok {
		tags = append(tags, strings.ToLower(language))
	}
	
	// Add type as tag
	if typ, ok := obj["type"].(string); ok {
		tags = append(tags, strings.ToLower(typ))
	}
	
	// Deduplicate
	seen := make(map[string]bool)
	unique := []string{}
	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			unique = append(unique, tag)
		}
	}
	
	return unique
}

func formatName(name string) string {
	// Convert kebab-case to Title Case
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.Title(part)
		}
	}
	return strings.Join(parts, " ")
}