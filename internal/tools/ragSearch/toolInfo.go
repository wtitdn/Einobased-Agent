package ragSearch

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ToolName = "query_chroma"
)

type QueryInput struct {
	Query      string `json:"query" jsonschema:"required" jsonschema_description:"The user's search query."`
	Collection string `json:"collection,omitempty" jsonschema_description:"Optional Chroma collection name, for example medical_knowledge or web_derived."`
	TopK       int    `json:"top_k,omitempty" jsonschema_description:"Maximum number of documents to return. Defaults to 5."`
}

type SearchResult struct {
	ID         int               `json:"id"`
	Collection string            `json:"collection"`
	Document   string            `json:"document"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Score      float64           `json:"score"`
}

type QueryOutput struct {
	Query       string         `json:"query"`
	Collections []string       `json:"collections"`
	Results     []SearchResult `json:"results"`
}

func NewTool() (tool.BaseTool, error) {
	return utils.InferTool(
		ToolName,
		"Search the local persisted Chroma database by vector similarity and return relevant documents with metadata.",
		func(ctx context.Context, input QueryInput) (QueryOutput, error) {
			return Query(ctx, input)
		},
	)
}

func Query(ctx context.Context, input QueryInput) (QueryOutput, error) {
	input.Query = strings.TrimSpace(input.Query)
	input.Collection = strings.TrimSpace(input.Collection)
	if input.Query == "" {
		return QueryOutput{}, fmt.Errorf("query is required")
	}
	if input.TopK <= 0 {
		input.TopK = 5
	}
	if input.TopK > 20 {
		input.TopK = 20
	}

	cfg, err := loadSearchConfig()
	if err != nil {
		return QueryOutput{}, err
	}
	queryVectors, err := queryToVector(ctx, input.Query, cfg)
	if err != nil {
		return QueryOutput{}, err
	}
	if len(queryVectors) == 0 || len(queryVectors[0]) == 0 {
		return QueryOutput{}, fmt.Errorf("query embedding is empty")
	}

	dbPath, err := databaseFilePath()
	if err != nil {
		return QueryOutput{}, err
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(dbPath)))
	if err != nil {
		return QueryOutput{}, err
	}
	defer db.Close()

	collections, err := listCollections(ctx, db)
	if err != nil {
		return QueryOutput{}, err
	}

	results, err := vectorSearch(ctx, queryVectors[0], input.TopK, input.Collection)
	if err != nil {
		return QueryOutput{}, err
	}

	return QueryOutput{
		Query:       input.Query,
		Collections: collections,
		Results:     results,
	}, nil
}

func databaseFilePath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to locate ragSearch package")
	}
	return filepath.Join(filepath.Dir(file), "database", "chroma.sqlite3"), nil
}

func listCollections(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "select name from collections order by name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []string
	for rows.Next() {
		var collection string
		if err := rows.Scan(&collection); err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}
	return collections, rows.Err()
}

func fetchMetadata(ctx context.Context, db *sql.DB, embeddingID int) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `
select key, string_value, int_value, float_value, bool_value
from embedding_metadata
where id = ?
order by key`, embeddingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string)
	for rows.Next() {
		var (
			key         string
			stringValue sql.NullString
			intValue    sql.NullInt64
			floatValue  sql.NullFloat64
			boolValue   sql.NullBool
		)
		if err := rows.Scan(&key, &stringValue, &intValue, &floatValue, &boolValue); err != nil {
			return nil, err
		}

		switch {
		case stringValue.Valid:
			metadata[key] = stringValue.String
		case intValue.Valid:
			metadata[key] = fmt.Sprintf("%d", intValue.Int64)
		case floatValue.Valid:
			metadata[key] = fmt.Sprintf("%g", floatValue.Float64)
		case boolValue.Valid:
			metadata[key] = fmt.Sprintf("%t", boolValue.Bool)
		}
	}
	if len(metadata) == 0 {
		return nil, rows.Err()
	}
	return metadata, rows.Err()
}
