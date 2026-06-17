package ragSearch

import (
	"context"
	"einoproject/internal/config"

	openaiembed "github.com/cloudwego/eino-ext/libs/acl/openai"
)

func NewEmbedder(ctx context.Context, cfg config.Config) (*openaiembed.EmbeddingClient, error) {
	dim := 1024
	embedder, err := openaiembed.NewEmbeddingClient(ctx, &openaiembed.EmbeddingConfig{
		APIKey:     cfg.Embed.Apikey,
		Model:      cfg.Embed.Model,
		BaseURL:    cfg.Embed.BaseURL,
		Dimensions: &dim,
	})
	if err != nil {
		return nil, err
	}
	return embedder, err
}
