package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

//go:embed knowledge/data
var data embed.FS

//go:embed index.html
var indexHTML string

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
	
	// Parse and execute template
	tmpl, err := template.New("index").Parse(indexHTML)
	if err != nil {
		http.Error(w, "Template error", 500)
		return
	}
	
	data := struct {
		ItemCount int
	}{
		ItemCount: len(knowledgeIndex),
	}
	
	tmpl.Execute(w, data)
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
	nameLower := strings.ToLower(knowledge.Name)
	
	// Track how many terms match and individual scores
	matchedTerms := 0
	totalScore := 0.0
	
	// Parse JSON to get framework and language
	var obj map[string]interface{}
	json.Unmarshal(knowledge.Content, &obj)
	framework := ""
	language := ""
	description := ""
	
	if f, ok := obj["framework"].(string); ok {
		framework = strings.ToLower(f)
	}
	if l, ok := obj["language"].(string); ok {
		language = strings.ToLower(l)
	}
	if d, ok := obj["description"].(string); ok {
		description = strings.ToLower(d)
	}
	
	for _, term := range terms {
		termMatched := false
		termScore := 0.0
		
		// Exact match in ID or name (highest priority)
		if strings.Contains(idLower, term) || strings.Contains(nameLower, term) {
			termScore += 3.0
			termMatched = true
		}
		
		// Framework match (very high priority for recipes)
		if framework != "" && strings.Contains(framework, term) {
			termScore += 2.5
			termMatched = true
		}
		
		// Language match (high priority)
		if language != "" && strings.Contains(language, term) {
			termScore += 2.0
			termMatched = true
		}
		
		// Description match (medium priority)
		if description != "" && strings.Contains(description, term) {
			termScore += 1.5
			termMatched = true
		}
		
		// General content match (low priority)
		if !termMatched && strings.Contains(contentStr, term) {
			termScore += 0.5
			termMatched = true
		}
		
		if termMatched {
			matchedTerms++
			totalScore += termScore
		}
	}
	
	// Calculate final score with bonuses for matching multiple terms
	if matchedTerms == 0 {
		return 0
	}
	
	// Base score from individual term scores
	score := totalScore / float64(len(terms))
	
	// Bonus for matching ALL terms (completeness bonus)
	if matchedTerms == len(terms) {
		score *= 2.0
	} else {
		// Penalty for missing terms
		score *= float64(matchedTerms) / float64(len(terms))
	}
	
	// Special bonus for exact framework matches (e.g., "flask" should rank Flask recipe highest)
	if knowledge.Type == "recipe" || knowledge.Type == "patterns" {
		for _, term := range terms {
			// If a term exactly matches the framework, huge bonus
			if framework != "" && framework == term {
				score += 5.0
			}
			// If the recipe name contains the term, bonus
			if strings.Contains(nameLower, term) {
				score += 2.0
			}
		}
	}
	
	return score
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