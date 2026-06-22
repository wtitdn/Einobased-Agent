package ragSearch

import (
	"context"
	"einoproject/internal/config"
	"net/http"
	"strings"

	openaiembed "github.com/cloudwego/eino-ext/libs/acl/openai"
)

func NewEmbedder(ctx context.Context, cfg config.Config) (*openaiembed.EmbeddingClient, error) {
	dim := 1024
	embedder, err := openaiembed.NewEmbeddingClient(ctx, &openaiembed.EmbeddingConfig{
		APIKey:     cfg.Embed.Apikey,
		Model:      cfg.Embed.Model,
		BaseURL:    normalizeEmbeddingBaseURL(cfg.Embed.BaseURL),
		HTTPClient: http.DefaultClient,
		Dimensions: &dim,
	})
	if err != nil {
		return nil, err
	}
	return embedder, err
}

func normalizeEmbeddingBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	for _, suffix := range []string{"/chat/completions", "/embeddings"} {
		baseURL = strings.TrimSuffix(baseURL, suffix)
	}
	return baseURL
}
