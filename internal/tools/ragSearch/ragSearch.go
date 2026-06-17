package ragSearch

import (
	"context"
	"database/sql"
	"einoproject/internal/config"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func queryToVector(ctx context.Context, query string, cfg config.Config) (embeddings [][]float64, err error) {
	embedder, err := NewEmbedder(ctx, cfg)
	if err != nil {
		return nil, err
	}
	vector, err := embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	return vector, err
}

func Search(ctx context.Context, query string) (message string, err error) {
	cfg, err := loadSearchConfig()
	if err != nil {
		return "", err
	}
	return SearchWithConfig(ctx, query, cfg)
}

func SearchWithConfig(ctx context.Context, query string, cfg config.Config) (message string, err error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	queryVectors, err := queryToVector(ctx, query, cfg)
	if err != nil {
		return "", err
	}
	if len(queryVectors) == 0 || len(queryVectors[0]) == 0 {
		return "", fmt.Errorf("query embedding is empty")
	}

	results, err := vectorSearch(ctx, queryVectors[0], 5, "")
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No relevant content was found in the local knowledge base.", nil
	}

	var builder strings.Builder
	builder.WriteString("Relevant content retrieved from the local Chroma knowledge base:\n\n")
	for i, result := range results {
		builder.WriteString(fmt.Sprintf("[%d] collection: %s, score: %.4f\n", i+1, result.Collection, result.Score))
		if len(result.Metadata) > 0 {
			builder.WriteString("metadata:\n")
			for key, value := range result.Metadata {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
			}
		}
		builder.WriteString("content:\n")
		builder.WriteString(strings.TrimSpace(result.Document))
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String()), nil
}

func loadSearchConfig() (config.Config, error) {
	candidates := []string{}
	if configPath := strings.TrimSpace(os.Getenv("CONFIG_PATH")); configPath != "" {
		candidates = append(candidates, configPath)
	}
	candidates = append(candidates, "config/config.yaml", "../config/config.yaml")

	var lastErr error
	for _, candidate := range candidates {
		cfg, err := config.Load(candidate)
		if err == nil {
			return cfg, nil
		}
		lastErr = err
	}
	return config.Config{}, fmt.Errorf("load rag search config: %w", lastErr)
}

func vectorSearch(ctx context.Context, queryVector []float64, topK int, collection string) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}
	collection = strings.TrimSpace(collection)

	dbPath, err := databaseFilePath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(dbPath)))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
select e.id, c.name, q.vector, d.string_value
from embeddings_queue q
join embeddings e on e.embedding_id = q.id
join segments s on s.id = e.segment_id
join collections c on c.id = s.collection
join embedding_metadata d on d.id = e.id and d.key = 'chroma:document'
where q.vector is not null`
	args := []any{}
	if collection != "" {
		query += " and c.name = ?"
		args = append(args, collection)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var (
			result SearchResult
			vector []byte
		)
		if err := rows.Scan(&result.ID, &result.Collection, &vector, &result.Document); err != nil {
			return nil, err
		}

		docVector, err := decodeFloat32Vector(vector)
		if err != nil {
			return nil, err
		}
		result.Score = cosineSimilarity(queryVector, docVector)
		result.Metadata, err = fetchMetadata(ctx, db, result.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func decodeFloat32Vector(data []byte) ([]float64, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid vector byte length: %d", len(data))
	}

	vector := make([]float64, len(data)/4)
	for i := range vector {
		bits := binary.LittleEndian.Uint32(data[i*4 : i*4+4])
		vector[i] = float64(math.Float32frombits(bits))
	}
	return vector, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
