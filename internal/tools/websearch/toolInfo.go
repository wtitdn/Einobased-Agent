package websearch

import (
	"context"
	"einoproject/internal/model"

	"github.com/cloudwego/eino-ext/components/tool/browseruse"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
)

const (
	ToolName = "browser_use"
)

func NewTool(ctx context.Context) (tool.BaseTool, error) {
	searchTool, err := duckduckgo.NewSearch(ctx, &duckduckgo.Config{
		MaxResults: 5,
		Region:     duckduckgo.RegionCN,
	})
	if err != nil {
		return nil, err
	}

	but, err := browseruse.NewBrowserUseTool(ctx, &browseruse.Config{
		Headless:         true,
		DDGSearchTool:    searchTool,
		ExtractChatModel: model.NewChatModel(),
	})
	if err != nil {
		return nil, err
	}
	return but, nil
}
